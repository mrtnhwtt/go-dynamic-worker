package queue

import (
	"fmt"
	"go-dynamic-worker/internal/models"
	"log/slog"
	"math/rand"
	"time"

	"github.com/tjarratt/babble"
)

type QueueService interface {
	PollQueue(maxMessage int32) ([]models.Message, error)
	DeleteMessage(id int32) error
}

type Queue struct {
	interval    time.Duration // interval between messages being added to the queue for polling
	temperature float32       // controls how often the queue will send the max number of message to a polling request
}

func NewQueue(interval time.Duration, temperature float32) (*Queue, error) {
	if temperature <= 0 || temperature > 1 {
		return nil, fmt.Errorf("temperature cannot be 0 or high than 1")
	}
	return &Queue{
		interval:    interval,
		temperature: temperature,
	}, nil
}

func (q *Queue) PollQueue(maxMessage int32) ([]models.Message, error) {
	slog.Info("============== Polling for messages ============== ")
	var messages []models.Message
	babbler := babble.NewBabbler()
	babbler.Count = 2

	nbMessage := biasedRandom(maxMessage, q.temperature)
	for id := range nbMessage {
		var inMessages []string
		nbMessages := rand.Intn(5) + 1
		for range nbMessages {
			body := babbler.Babble()
			inMessages = append(inMessages, body)
		}
		messages = append(messages, models.Message{
			ID:     id + 1,
			Events: inMessages,
		})
	}
	time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
	return messages, nil
}

func biasedRandom(x int32, t float32) int32 {
	if t < 0 || t > 1 {
		panic("t must be between 0.0 and 1.0")
	}

	if rand.Float32() < t {
		return x
	}
	return rand.Int31n(x) // returns 0 to x-1
}

func (q *Queue) DeleteMessage(id int32) error {
	return nil
}
