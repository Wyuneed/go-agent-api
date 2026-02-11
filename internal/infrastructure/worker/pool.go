package worker

import (
	"context"
	"sync"
)

type Job func(ctx context.Context) error

type Pool struct {
	workers int
	jobs    chan Job
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewPool(ctx context.Context, workers, queueSize int) *Pool {
	ctx, cancel := context.WithCancel(ctx)
	p := &Pool{
		workers: workers,
		jobs:    make(chan Job, queueSize),
		ctx:     ctx,
		cancel:  cancel,
	}

	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			job(p.ctx)
		}
	}
}

func (p *Pool) Submit(job Job) bool {
	select {
	case p.jobs <- job:
		return true
	case <-p.ctx.Done():
		return false
	default:
		return false
	}
}

func (p *Pool) SubmitWait(job Job) bool {
	select {
	case p.jobs <- job:
		return true
	case <-p.ctx.Done():
		return false
	}
}

func (p *Pool) Shutdown() {
	p.cancel()
	close(p.jobs)
	p.wg.Wait()
}
