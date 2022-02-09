package workflow

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/multierr"
)

type recoverTask struct {
	Task
}

func SafeTask(t Task) Task {
	return &recoverTask{
		Task: t,
	}
}

func (rt *recoverTask) Commit(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("found:%v,trace:%s", r, buf)
			}
		}
	}()
	err = rt.Task.Commit(ctx)
	return
}

func (rt *recoverTask) Rollback(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("found:%v,trace:%s", r, buf)
			}
		}
	}()
	err = rt.Task.Rollback(ctx)
	return
}

type retryTask struct {
	Task
	attempts int
	interval time.Duration
}

func RetryTask(t Task, opts ...Option) Task {
	o := &option{
		policy: PolicyRetry,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &retryTask{
		Task:     t,
		attempts: o.attempt,
		interval: o.interval,
	}
}

func (rt *retryTask) Commit(ctx context.Context) error {
	err := rt.Task.Commit(ctx)
	if err == nil {
		return nil
	}
	if rt.Task.Policy() == PolicyRetry {
		var tempAttempts int
		backOff := rt.newBackOff() // 退避算法 保证时间间隔为指数级增长
		currentInterval := 0 * time.Millisecond
		timer := time.NewTimer(currentInterval)
		for {
			select {
			case <-timer.C:
				shouldRetry := tempAttempts < rt.attempts
				if !shouldRetry {
					timer.Stop()
					return err
				}
				if err = rt.Task.Commit(ctx); err == nil {
					shouldRetry = false
				}
				if !shouldRetry {
					timer.Stop()
					return err
				}
				// 计算下一次
				currentInterval = backOff.NextBackOff()
				tempAttempts++
				// 定时器重置
				timer.Reset(currentInterval)
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}
	return err
}

func (rt *retryTask) newBackOff() backoff.BackOff {
	if rt.attempts < 2 || rt.interval <= 0 {
		return &backoff.ZeroBackOff{}
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = rt.interval

	// calculate the multiplier for the given number of attempts
	// so that applying the multiplier for the given number of attempts will not exceed 2 times the initial interval
	// it allows to control the progression along the attempts
	b.Multiplier = math.Pow(2, 1/float64(rt.attempts-1))

	// according to docs, b.Reset() must be called before using
	b.Reset()
	return b
}

type parallelTask struct {
	id     string
	name   string
	policy Policy

	ch    chan Task
	tasks []Task

	mux   sync.Mutex
	stack *stack

	cur     int
	errOnce sync.Once
	err     error
}

func ParallelTask(opts ...Option) Task {
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &option{
		name:   "spawn-task-" + uidStr,
		policy: PolicyRetry,
		tasks:  make([]Task, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &parallelTask{
		id:     uidStr,
		name:   o.name,
		policy: o.policy,
		tasks:  o.tasks,
		stack:  NewStack(),
	}
}

func (s *parallelTask) ID() string {
	return s.id
}

func (s *parallelTask) Name() string {
	return s.name
}

func (s *parallelTask) Policy() Policy {
	return s.policy
}

func (s *parallelTask) Commit(ctx context.Context) error {
	newCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, task := range s.tasks {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, t Task) {
			if err := t.Commit(ctx); err != nil {
				s.errOnce.Do(func() {
					s.err = err
					cancel()
				})
			}
			wg.Done()
		}(newCtx, &wg, cancel, task)
	}
	wg.Wait()
	cancel()
	return s.err
}

func (s *parallelTask) Rollback(ctx context.Context) error {
	panic("implement me")
}

type pipelineTask struct {
	id     string
	name   string
	policy Policy

	tasks []Task
	cur   int
}

func PipelineTask(opts ...Option) Task {
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &option{
		name:   "pipeline-task-" + uidStr,
		policy: PolicyRevert,
		tasks:  make([]Task, 0),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &pipelineTask{
		id:     uidStr,
		name:   o.name,
		policy: o.policy,
		tasks:  o.tasks,
	}
}

func (s *pipelineTask) ID() string {
	return s.id
}

func (s *pipelineTask) Name() string {
	return s.name
}

func (s *pipelineTask) Policy() Policy {
	return s.policy
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
