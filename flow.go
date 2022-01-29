package taskflow

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/crochee/lirity/id"
	"github.com/crochee/lirity/log"
)

type pipelineFlow struct {
	name     string
	tasks    []Task
	notifier Notifier
	policy   Policy
	attempts int
	interval time.Duration
}

func NewPipelineFlow(opts ...Option) Flow {
	o := &option{
		name:     "pipeline-Flow-" + id.UUID(),
		notifier: NoneNotify{},
		policy:   PolicyRetry,
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
	policy := p.policy
	for _, task := range p.tasks {
		tempPolicy := task.Policy()
		if policy < tempPolicy {
			policy = tempPolicy
		}
		if err := execute(ctx, task, p.notifier, policy, p.attempts, p.interval); err != nil {
			return err
		}
	}
	return nil
}

type spawnFlow struct {
	name     string
	tasks    []Task
	index    int
	notifier Notifier
	policy   Policy
	attempts int
	interval time.Duration

	cancel  func()
	wg      sync.WaitGroup
	errOnce sync.Once
	err     error
}

func NewSpawnFlow(opts ...Option) Flow {
	o := &option{
		name:     "spawn-Flow-" + id.UUID(),
		notifier: NoneNotify{},
		policy:   PolicyRetry,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &spawnFlow{
		name:     o.name,
		tasks:    make([]Task, 0, 2),
		notifier: o.notifier,
		policy:   o.policy,
		attempts: o.attempt,
		interval: o.interval,
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
	newCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, task := range s.tasks {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, t Task) {
			if err := execute(ctx, t, s.notifier, s.policy, s.attempts, s.interval); err != nil {
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

func newBackOff(attempts int, interval time.Duration) backoff.BackOff {
	if attempts < 2 || interval <= 0 {
		return &backoff.ZeroBackOff{}
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = interval

	// calculate the multiplier for the given number of attempts
	// so that applying the multiplier for the given number of attempts will not exceed 2 times the initial interval
	// it allows to control the progression along the attempts
	b.Multiplier = math.Pow(2, 1/float64(attempts-1))

	// according to docs, b.Reset() must be called before using
	b.Reset()
	return b
}

func execute(ctx context.Context, t Task, notifier Notifier, policy Policy, attempts int, interval time.Duration) error {
	taskName := t.Name()
	notifier.Event(ctx, fmt.Sprintf("task called %s starts running", taskName))
	notifier.Notify(ctx, taskName, 0)

	err := t.Commit(ctx)
	if err == nil {
		notifier.Event(ctx, fmt.Sprintf("task called %s ends running", taskName))
		notifier.Notify(ctx, taskName, 100)
		return nil
	}
	notifier.Event(ctx, fmt.Sprintf("task called %s commits appears %e,and want to rollback", taskName, err))

	if policy == PolicyRetry {
		var (
			tempAttempts int
			breakFlag    bool
		)
		backOff := newBackOff(attempts, interval) // 退避算法 保证时间间隔为指数级增长
		currentInterval := 0 * time.Millisecond
		timer := time.NewTimer(currentInterval)
		for {
			select {
			case <-timer.C:
				shouldRetry := tempAttempts < attempts
				if err = t.Commit(ctx); err == nil {
					shouldRetry = false
				}
				log.FromContext(ctx).Warnf("task called %s runs for the %d time,and commit appears %e",
					taskName, tempAttempts, err)
				if !shouldRetry {
					timer.Stop()
					breakFlag = true
					break
				}
				// 计算下一次
				currentInterval = backOff.NextBackOff()
				tempAttempts++
				// 定时器重置
				timer.Reset(currentInterval)
			case <-ctx.Done():
				timer.Stop()
				err = ctx.Err()
				breakFlag = true
			}
			if breakFlag {
				break
			}
		}
	}
	if err == nil {
		notifier.Event(ctx, fmt.Sprintf("task called %s ends running", taskName))
		notifier.Notify(ctx, taskName, 100)
		return nil
	}
	notifier.Event(ctx, fmt.Sprintf("task called %s commits appears %e,and want to rollback", taskName, err))

	err = t.Rollback(ctx)
	notifier.Event(ctx, fmt.Sprintf("task called %s rollbacks appears %e", taskName, err))
	notifier.Notify(ctx, taskName, 0)
	return err
}
