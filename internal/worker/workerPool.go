package worker

import (
	"context"
	"go-dynamic-worker/internal/models"
	"go-dynamic-worker/internal/queue"
	"go-dynamic-worker/internal/sender"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

type WorkerPoolService interface {
	Init() error
	SubmitMessage(msg []models.Message)
	Stop()
}

type WorkerPool struct {
	workers     []WorkerService
	ctx         context.Context
	wg          sync.WaitGroup // WaitGroup to track that each worker finished and close the channel when the main context cancels
	jobChan     chan models.JobRequest
	stopWorkers context.CancelFunc
	sem         *semaphore.Weighted
}

func NewWorkerPool(ctx context.Context, workerCount int, ss sender.SenderService, qs queue.QueueService) *WorkerPool {
	workerCtx, cancel := context.WithCancel(ctx)
	pool := &WorkerPool{
		workers:     make([]WorkerService, workerCount),
		ctx:         ctx,
		wg:          sync.WaitGroup{},
		jobChan:     make(chan models.JobRequest, workerCount*2),
		stopWorkers: cancel,
		sem:         semaphore.NewWeighted(int64(workerCount)),
	}

	for i := range workerCount {
		pool.workers[i] = NewWorker(workerCtx, i+1, ss, qs, pool.jobChan)
	}

	return pool
}

func (wp *WorkerPool) Init() error {
	releaseWaitGroup := func() { // use a closure to give access to the worker to the wg.Done function without passing the waitgroup
		wp.wg.Done()
	}
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go worker.Run(releaseWaitGroup)
	}
	return nil
}

func (wp *WorkerPool) Stop() {
	slog.Info("Stopping worker pool...")

	wp.stopWorkers()

	slog.Info("Waiting for workers to finish current job...")

	wp.wg.Wait()

	slog.Info("All worker stopped, closing job receiving channel...")

	close(wp.jobChan)

	slog.Info("Worker pool stop completed")
}

func (wp *WorkerPool) SubmitMessage(messages []models.Message) {
	slog.Info("WorkePool received message to assign to workers", "messageCount", len(messages))
	for _, msg := range messages {
		select {
		default:
			jobCtx, jobClose := context.WithTimeout(context.Background(), 120*time.Second)
			resultCh := make(chan error, 1)
			jobReq := models.JobRequest{
				Message: msg,
				Result:  resultCh,
			}
			wp.sem.Acquire(jobCtx, 1) // acquire slot
			wp.jobChan <- jobReq

			cleanup := func() {
				close(resultCh)
				jobClose()
			}

			go wp.HandleJobResult(msg.ID, resultCh, cleanup)

		case <-wp.ctx.Done():
			slog.Warn("SubmitMessage context cancelled")
			return
		}
	}
	slog.Info("WorkPool sent all message to be picked up by workers. Returning to caller.")
}

func (wp *WorkerPool) HandleJobResult(msgID int32, result <-chan error, cleanup func()) {
	err := <-result
	if err != nil {
		slog.Error("Worker failed to process message", "ID", msgID, "error", err)
	} else {
		slog.Info("Message processed successfully", "ID", msgID)
	}
	wp.sem.Release(1)
	cleanup()
}
