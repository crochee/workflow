package taskflow

import (
	"context"
	"encoding/hex"
	"math"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	uuid "github.com/satori/go.uuid"
)

type pipelineFlow struct {
	id       string
	name     string
	tasks    []Task
	notifier Notifier
	policy   Policy
	attempts int
	interval time.Duration
}

func NewPipelineFlow(opts ...Option) Flow {
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &option{
		name:     "pipeline-flow-" + uidStr,
		notifier: NoneNotify{},
		policy:   PolicyRetry,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &pipelineFlow{
		id:       uidStr,
		name:     o.name,
		tasks:    make([]Task, 0, 2),
		notifier: o.notifier,
		policy:   o.policy,
		attempts: o.attempt,
		interval: o.interval,
	}
}

func (p *pipelineFlow) ID() string {
	return p.name
}

func (p *pipelineFlow) Name() string {
	return p.name
}

func (p *pipelineFlow) Add(tasks ...Task) Flow {
	p.tasks = append(p.tasks, tasks...)
	return p
}

func (p *pipelineFlow) ListTask() []Task {
	return p.tasks
}

func (p *pipelineFlow) Run(ctx context.Context) error {
	var applyAll bool
	for _, task := range p.tasks {
		policy := task.Policy()
		if policy < p.policy {
			policy = p.policy
		}
		if policy == PolicyRevertAll {
			applyAll = true
		}
		if applyAll {
			policy = PolicyRevertAll
		}
		if err := execute(ctx, task, p.notifier, policy, p.attempts, p.interval); err != nil {
			return err
		}
	}
	return nil
}

type spawnFlow struct {
	id       string
	name     string
	tasks    []Task
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
	uid := uuid.NewV1()
	uidStr := hex.EncodeToString(uid[:])
	o := &option{
		name:     "spawn-flow-" + uidStr,
		notifier: NoneNotify{},
		policy:   PolicyRetry,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &spawnFlow{
		id:       uidStr,
		name:     o.name,
		tasks:    make([]Task, 0, 2),
		notifier: o.notifier,
		policy:   o.policy,
		attempts: o.attempt,
		interval: o.interval,
	}
}

func (s *spawnFlow) ID() string {
	return s.id
}

func (s *spawnFlow) Name() string {
	return s.name
}

func (s *spawnFlow) Add(tasks ...Task) Flow {
	s.tasks = append(s.tasks, tasks...)
	return s
}

func (s *spawnFlow) ListTask() []Task {
	return s.tasks
}

func (s *spawnFlow) Run(ctx context.Context) error {
	newCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, task := range s.tasks {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc, t Task) {
			policy := t.Policy()
			if policy < s.policy {
				policy = s.policy
			}
			if err := execute(ctx, t, s.notifier, policy, s.attempts, s.interval); err != nil {
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
	notifier.Event(ctx, "task called %s starts running", taskName)
	notifier.Notify(ctx, taskName, 0)

	err := t.Commit(ctx)
	if err == nil {
		notifier.Event(ctx, "task called %s ends running", taskName)
		notifier.Notify(ctx, taskName, 100)
		return nil
	}
	notifier.Event(ctx, "task called %s commits appears %v,and want to rollback", taskName, err)

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
				notifier.Event(ctx, "task called %s runs for the %d time,and commit appears %v",
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
		notifier.Event(ctx, "task called %s ends running", taskName)
		notifier.Notify(ctx, taskName, 100)
		return nil
	}
	notifier.Event(ctx, "task called %s commits appears %v,and want to rollback", taskName, err)

	err = t.Rollback(ctx)
	notifier.Event(ctx, "task called %s rollbacks appears %v", taskName, err)
	notifier.Notify(ctx, taskName, 0)
	return err
}
