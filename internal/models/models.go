package models

import "context"

type Message struct {
	ID     string
	Events []string
}

type JobRequest struct {
	Message Message
	JobCtx  context.Context
	Result  chan<- error // were to report completion
}
