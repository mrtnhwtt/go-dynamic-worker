package worker

import (
	"context"
	"fmt"
	"go-dynamic-worker/internal/models"
	"go-dynamic-worker/internal/queue"
	"go-dynamic-worker/internal/sender"
	"log/slog"
	"time"
)

type WorkerService interface {
	Run(close func())
}

type Worker struct {
	id      int
	ctx     context.Context
	sender  sender.SenderService
	queue   queue.QueueService
	jobChan chan models.JobRequest
}

func NewWorker(ctx context.Context, id int, ss sender.SenderService, qs queue.QueueService, jobChan chan models.JobRequest) *Worker {
	return &Worker{
		id:      id,
		ctx:     ctx,
		sender:  ss,
		queue:   qs,
		jobChan: jobChan,
	}
}

func (w *Worker) Run(exitWorker func()) {
	slog.Info("Started worker Run", "workerID", w.id)
	for { // looping until cancellation is received, when receiving a job creates a new goroutine
		select {
		case <-w.ctx.Done():
			slog.Info("Cancellation request received, won't accept any more jobs.", "WorkerID", w.id)
			exitWorker()
			return
		case jobReq := <-w.jobChan:
			go func(job models.JobRequest) {
				slog.Info("Received job request", "workerID", w.id, "MessageID", job.Message.ID)
				job.Result <- w.HandleMessage(job.JobCtx, job.Message)
			}(jobReq)
		}
	}
}

func (w *Worker) HandleMessage(jobCtx context.Context, messages models.Message) error {
	slog.Info("Handling job", "workerID", w.id, "messageID", messages.ID)
	for i, eventMsg := range messages.Events {
		slog.Info("Processing event...", "workerID", w.id, "eventID", i+1, "messageID", messages.ID)
		time.Sleep(1 * time.Second) // simulate some processing
		slog.Info("Finished processing event, sending...", "workerID", w.id, "eventID", i+1, "messageID", messages.ID)
		err := w.sender.Send(jobCtx, eventMsg)
		if err != nil {
			return fmt.Errorf("Failed to send all messages")
		}
	}
	slog.Info("Finished proccesing all events in message, deleting from Queue...", "workerID", w.id, "messageID", messages.ID)
	if err := w.queue.DeleteMessage(messages.ID); err != nil {
		slog.Error("Failed to delete message", "workerID", w.id, "messageID", messages.ID)
		return err
	}
	slog.Info("Message deleted from queue", "workerID", w.id, "messageID", messages.ID)
	return nil
}
