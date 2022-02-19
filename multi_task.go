package workflow

import (
	"context"
	"encoding/hex"
	"sync"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/multierr"
)

type multiOption struct {
	name  string
	tasks []Task
}

type MultiOption func(*multiOption)

func WithName(name string) MultiOption {
	return func(o *multiOption) {
		o.name = name
	}
}

func WithTasks(tasks ...Task) MultiOption {
	return func(o *multiOption) {
		o.tasks = tasks
	}
}

type parallelTask struct {
	id   string
	name string

	executedTasks []Task
	mutex         sync.Mutex

	tasks []Task

	errOnce sync.Once
	err     error
}

func ParallelTask(opts ...MultiOption) Task {
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &multiOption{
		name:  "parallel-task-" + uidStr,
		tasks: make([]Task, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &parallelTask{
		id:    uidStr,
		name:  o.name,
		tasks: o.tasks,
	}
}

func (s *parallelTask) ID() string {
	return s.id
}

func (s *parallelTask) Name() string {
	return s.name
}

func (s *parallelTask) Commit(ctx context.Context) error {
	newCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, task := range s.tasks {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, t Task) {
			select {
			case <-ctx.Done():
			default:
				if err := t.Commit(ctx); err != nil {
					s.errOnce.Do(func() {
						s.err = err
						cancel()
					})
				}
				s.mutex.Lock()
				s.executedTasks = append(s.executedTasks, t)
				s.mutex.Unlock()
			}
			wg.Done()
		}(newCtx, &wg, cancel, task)
	}
	wg.Wait()
	cancel()
	return s.err
}

func (s *parallelTask) Rollback(ctx context.Context) error {
	s.err = nil
	var wg sync.WaitGroup
	for _, task := range s.executedTasks {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, t Task) {
			var err error
			select {
			case <-ctx.Done():
				err = ctx.Err()
			default:
				err = t.Rollback(ctx)
			}
			s.mutex.Lock()
			s.err = multierr.Append(s.err, err)
			s.mutex.Unlock()
			wg.Done()
		}(ctx, &wg, task)
	}
	wg.Wait()
	return s.err
}

type pipelineTask struct {
	id   string
	name string

	tasks []Task
	cur   int
}

func PipelineTask(opts ...MultiOption) Task {
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &multiOption{
		name:  "pipeline-task-" + uidStr,
		tasks: make([]Task, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &pipelineTask{
		id:    uidStr,
		name:  o.name,
		tasks: o.tasks,
	}
}

func (s *pipelineTask) ID() string {
	return s.id
}

func (s *pipelineTask) Name() string {
	return s.name
}

func (s *pipelineTask) Commit(ctx context.Context) error {
	for index, task := range s.tasks {
		if err := task.Commit(ctx); err != nil {
			s.cur = index
			return err
		}
	}
	return nil
}

func (s *pipelineTask) Rollback(ctx context.Context) error {
	var err error
	for i := s.cur; i >= 0; i-- {
		err = multierr.Append(err, s.tasks[i].Rollback(ctx))
	}
	return err
}
