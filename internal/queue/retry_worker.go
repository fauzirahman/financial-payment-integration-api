package queue

import (
	"context"
	"fmt"
	"time"
)

type Job struct {
	ID          string
	Payload     any
	Attempt     int
	MaxAttempts int
	NextRunAt   time.Time
	CreatedAt   time.Time
}

type RetryProcessor interface {
	Process(context.Context, Job) error
}

type RetryWorker struct {
	processor RetryProcessor
}

func NewRetryWorker(processor RetryProcessor) *RetryWorker {
	return &RetryWorker{processor: processor}
}

func (w *RetryWorker) Process(ctx context.Context, job Job) error {
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 3
	}
	if job.Attempt >= job.MaxAttempts {
		return fmt.Errorf("job %s exhausted retries", job.ID)
	}
	if err := w.processor.Process(ctx, job); err != nil {
		job.Attempt++
		job.NextRunAt = time.Now().Add(time.Second * time.Duration(job.Attempt))
		return err
	}
	return nil
}
