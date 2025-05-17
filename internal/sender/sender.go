package sender

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
)

type SenderService interface {
	Send(ctx context.Context, msg string) error
}
type Sender struct {
}

func NewSender() *Sender {
	return &Sender{}
}

func (s *Sender) Send(ctx context.Context, msg string) error {
	if rand.Intn(100) < 1 {
		return fmt.Errorf("failed to send")
	}
	slog.Info("Successfully sent message")
	return nil
}
