package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryQueue simulates a simple work queue with retry attempts and a dead-letter queue.
type InMemoryQueue struct {
	mu          sync.Mutex
	jobs        []Job
	deadLetter  []Job
	retryWorker *RetryWorker
}

func NewInMemoryQueue(processor RetryProcessor) *InMemoryQueue {
	return &InMemoryQueue{
		jobs:        make([]Job, 0),
		deadLetter:  make([]Job, 0),
		retryWorker: NewRetryWorker(processor),
	}
}

func (q *InMemoryQueue) Enqueue(job Job) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.NextRunAt.IsZero() {
		job.NextRunAt = job.CreatedAt
	}
	q.jobs = append(q.jobs, job)
}

func (q *InMemoryQueue) Consume(ctx context.Context) error {
	q.mu.Lock()
	jobs := append([]Job(nil), q.jobs...)
	q.jobs = nil
	q.mu.Unlock()

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().Before(job.NextRunAt) {
			q.mu.Lock()
			q.jobs = append(q.jobs, job)
			q.mu.Unlock()
			continue
		}

		if err := q.retryWorker.Process(ctx, job); err != nil {
			if job.Attempt >= job.MaxAttempts {
				q.mu.Lock()
				q.deadLetter = append(q.deadLetter, job)
				q.mu.Unlock()
				continue
			}
			job.Attempt++
			job.NextRunAt = time.Now().Add(time.Second * time.Duration(job.Attempt))
			q.mu.Lock()
			q.jobs = append(q.jobs, job)
			q.mu.Unlock()
			continue
		}
	}
	return nil
}

func (q *InMemoryQueue) DeadLetter() []Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]Job(nil), q.deadLetter...)
}

func (q *InMemoryQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

func (q *InMemoryQueue) ExampleProcessor(ctx context.Context, job Job) error {
	_ = ctx
	if job.Attempt > 0 {
		return fmt.Errorf("transient failure")
	}
	return nil
}
