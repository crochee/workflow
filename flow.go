package taskflow

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/crochee/lirity/id"
)

type pipelineFlow struct {
	name  string
	tasks []Task
	index int
}

func NewPipelineFlow(opts ...Option) Flow {
	o := &option{
		name: "pipeline-flow-" + id.UUID(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &pipelineFlow{
		name:  o.name,
		tasks: make([]Task, 0, 2),
	}
}

func (p *pipelineFlow) Name() string {
	return p.name
}

func (p *pipelineFlow) Add(tasks ...Task) Flow {
	p.tasks = append(p.tasks, tasks...)
	return p
}

func (p *pipelineFlow) Run(ctx context.Context) error {
	for _, task := range p.tasks {
		if err := task.Execute(ctx); err != nil {
			return err
		}
	}
	return nil
}

type spawnFlow struct {
	name  string
	tasks []Task

	cancel func()

	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
}

func NewSpawnFlow(opts ...Option) Flow {
	o := &option{
		name: "spawn-flow-" + id.UUID(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &spawnFlow{
		name:  o.name,
		tasks: make([]Task, 0, 2),
	}
}

func (s *spawnFlow) Name() string {
	return s.name
}

func (s *spawnFlow) Add(tasks ...Task) Flow {
	s.tasks = append(s.tasks, tasks...)
	return s
}

func (s *spawnFlow) Run(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)
	for _, task := range s.tasks {
		s.wg.Add(1)
		go func(t Task) {
			var err error
			defer func() {
				if r := recover(); r != nil {
					var ok bool
					if err, ok = r.(error); !ok {
						err = fmt.Errorf("%v.Stack:%s", r, debug.Stack())
					}
				}
				if err != nil {
					s.errOnce.Do(func() {
						s.err = err
						s.cancel()
					})
				}
				s.wg.Done()
			}()
			err = t.Execute(ctx)
		}(task)
	}
	s.wg.Wait()
	if s.cancel != nil {
		s.cancel()
	}
	return s.err
}
