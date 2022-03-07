package workflow

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/multierr"
)

type parallelTask struct {
	id          string
	name        string
	state       State
	description string
	createTime  time.Time
	updateTime  time.Time
	meta        map[string]interface{}
	callbacks   []Callback
	now         func() time.Time

	tasks         []Task
	executedTasks []Task
	rwMutex       sync.RWMutex
	errOnce       sync.Once
	err           error
}

func ParallelTask(opts ...TaskOption) Task {
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &taskOption{
		name:        "parallel-task-" + uidStr,
		description: "it's a parallel task",
		meta:        map[string]interface{}{},
		callbacks:   make([]Callback, 0),
		nowFunc:     time.Now,
		tasks:       make([]Task, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	now := o.nowFunc()
	return &parallelTask{
		id:            uidStr,
		name:          o.name,
		state:         Ready,
		description:   o.description,
		createTime:    now,
		updateTime:    now,
		meta:          o.meta,
		callbacks:     o.callbacks,
		now:           o.nowFunc,
		tasks:         o.tasks,
		executedTasks: nil,
		rwMutex:       sync.RWMutex{},
		errOnce:       sync.Once{},
		err:           nil,
	}
}

func (p *parallelTask) ID() string {
	return p.id
}

func (p *parallelTask) Name() string {
	return p.name
}

func (p *parallelTask) State() (State, error) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()
	return p.state, p.err
}

func (p *parallelTask) Description() string {
	return p.description
}

func (p *parallelTask) CreateTime() time.Time {
	return p.createTime
}

func (p *parallelTask) UpdateTime() time.Time {
	return p.createTime
}

func (p *parallelTask) Meta() map[string]interface{} {
	return p.meta
}

func (p *parallelTask) Commit(ctx context.Context, opts ...TaskOption) error {
	o := &taskOption{}
	for _, opt := range opts {
		opt(o)
	}
	callbacks := append(p.callbacks, o.callbacks...)

	p.rwMutex.Lock()
	p.state = Running
	p.updateTime = p.now()
	p.rwMutex.Unlock()

	for _, callback := range callbacks {
		callback.Trigger(ctx, p)
	}

	newCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, task := range p.tasks {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, t Task) {
			select {
			case <-ctx.Done():
			default:
				if err := t.Commit(ctx); err != nil {
					p.errOnce.Do(func() {
						p.rwMutex.Lock()
						p.err = err
						p.rwMutex.Unlock()
						cancel()
					})
				}
				p.rwMutex.Lock()
				p.executedTasks = append(p.executedTasks, t)
				p.rwMutex.Unlock()
			}
			wg.Done()
		}(newCtx, &wg, cancel, task)
	}
	wg.Wait()
	cancel()

	p.rwMutex.Lock()
	if p.err != nil {
		p.state = Error
	} else {
		p.state = Success
	}
	p.updateTime = p.now()
	p.rwMutex.Unlock()

	for _, callback := range callbacks {
		callback.Trigger(ctx, p)
	}
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()
	return p.err
}

func (p *parallelTask) Rollback(ctx context.Context, opts ...TaskOption) error {
	p.rwMutex.Lock()
	p.err = nil
	p.rwMutex.Unlock()

	var wg sync.WaitGroup
	for _, task := range p.executedTasks {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, t Task) {
			var err error
			select {
			case <-ctx.Done():
				err = ctx.Err()
			default:
				err = t.Rollback(ctx)
			}
			p.rwMutex.Lock()
			p.err = multierr.Append(p.err, err)
			p.rwMutex.Unlock()
			wg.Done()
		}(ctx, &wg, task)
	}
	wg.Wait()
	p.rwMutex.RLock()
	if p.err != nil {
		p.rwMutex.RUnlock()

		p.rwMutex.Lock()
		p.state = Error
		p.updateTime = p.now()
		p.rwMutex.Unlock()

		o := &taskOption{}
		for _, opt := range opts {
			opt(o)
		}
		callbacks := append(p.callbacks, o.callbacks...)
		for _, callback := range callbacks {
			callback.Trigger(ctx, p)
		}
		p.rwMutex.RLock()
		defer p.rwMutex.RUnlock()
		return p.err
	}
	p.rwMutex.RUnlock()
	return nil
}

type pipelineTask struct {
	id          string
	name        string
	state       State
	description string
	createTime  time.Time
	updateTime  time.Time
	meta        map[string]interface{}
	callbacks   []Callback
	now         func() time.Time

	tasks   []Task
	cur     int
	err     error
	rwMutex sync.RWMutex
}

func PipelineTask(opts ...TaskOption) Task {
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &taskOption{
		name:        "pipeline-task-" + uidStr,
		description: "it's a pipeline task",
		meta:        map[string]interface{}{},
		callbacks:   make([]Callback, 0),
		nowFunc:     time.Now,
		tasks:       make([]Task, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	now := o.nowFunc()
	return &pipelineTask{
		id:          uidStr,
		name:        o.name,
		state:       Ready,
		description: o.description,
		createTime:  now,
		updateTime:  now,
		meta:        o.meta,
		callbacks:   o.callbacks,
		now:         o.nowFunc,
		tasks:       o.tasks,
		cur:         0,
		err:         nil,
		rwMutex:     sync.RWMutex{},
	}
}

func (s *pipelineTask) ID() string {
	return s.id
}

func (s *pipelineTask) Name() string {
	return s.name
}

func (s *pipelineTask) State() (State, error) {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.state, s.err
}

func (s *pipelineTask) Description() string {
	return s.description
}

func (s *pipelineTask) CreateTime() time.Time {
	return s.createTime
}

func (s *pipelineTask) UpdateTime() time.Time {
	return s.createTime
}

func (s *pipelineTask) Meta() map[string]interface{} {
	return s.meta
}

func (s *pipelineTask) Commit(ctx context.Context, opts ...TaskOption) error {
	o := &taskOption{}
	for _, opt := range opts {
		opt(o)
	}
	callbacks := append(s.callbacks, o.callbacks...)

	s.rwMutex.Lock()
	s.state = Running
	s.updateTime = s.now()
	s.rwMutex.Unlock()

	for _, callback := range callbacks {
		callback.Trigger(ctx, s)
	}
	var err error
	for index, task := range s.tasks {
		if err = task.Commit(ctx, opts...); err != nil {
			s.cur = index
			break
		}
	}

	s.rwMutex.Lock()
	if err != nil {
		s.state = Error
		s.err = err
	} else {
		s.state = Success
	}
	s.updateTime = s.now()
	s.rwMutex.Unlock()

	for _, callback := range callbacks {
		callback.Trigger(ctx, s)
	}
	return err
}

func (s *pipelineTask) Rollback(ctx context.Context, opts ...TaskOption) error {
	var err error
	for i := s.cur; i >= 0; i-- {
		err = multierr.Append(err, s.tasks[i].Rollback(ctx))
	}
	if err != nil {
		s.rwMutex.Lock()
		s.state = Error
		s.updateTime = s.now()
		s.rwMutex.Unlock()

		o := &taskOption{}
		for _, opt := range opts {
			opt(o)
		}
		callbacks := append(s.callbacks, o.callbacks...)
		for _, callback := range callbacks {
			callback.Trigger(ctx, s)
		}
	}
	return err
}
