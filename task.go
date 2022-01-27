package taskflow

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/crochee/lirity/log"
)

type FuncJob func(ctx context.Context) error

func (f FuncJob) Execute(ctx context.Context) error { return f(ctx) }

type commitTask struct {
	c        Committer
	notifier Notifier
}

func NewCommitTask(c Committer, opts ...Option) Task {
	o := &option{
		notifier: NoneNotify{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return &commitTask{c: c, notifier: o.notifier}
}

func (c *commitTask) Execute(ctx context.Context) error {
	name := c.c.Name()
	c.notifier.Event(ctx, fmt.Sprintf("task called %s starts running", name))
	c.notifier.Notify(ctx, name, 0)

	err := c.c.Commit(ctx)
	if err == nil {
		c.notifier.Event(ctx, fmt.Sprintf("task called %s ends running", name))
		c.notifier.Notify(ctx, name, 100)
		return nil
	}

	c.notifier.Event(ctx, fmt.Sprintf("task called %s commits appears %e,and want to rollback", name, err))

	err = c.c.Rollback(ctx)

	c.notifier.Event(ctx, fmt.Sprintf("task called %s rollbacks appears %e", name, err))
	c.notifier.Notify(ctx, name, 0)
	return err
}

type retryTask struct {
	c        Committer
	notifier Notifier
	policy   Policy
	attempts int
	interval time.Duration
}

func NewRetryTask(c Committer, policy Policy, opts ...Option) Task {
	o := &option{
		notifier: NoneNotify{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return &retryTask{
		c:        c,
		policy:   policy,
		attempts: o.attempt,
		interval: o.interval,
		notifier: o.notifier,
	}
}

func (r *retryTask) Execute(ctx context.Context) error {
	name := r.c.Name()
	r.notifier.Event(ctx, fmt.Sprintf("task called %s starts running", name))
	r.notifier.Notify(ctx, name, 0)

	err := r.c.Commit(ctx)
	if err == nil {
		r.notifier.Event(ctx, fmt.Sprintf("task called %s ends running", name))
		r.notifier.Notify(ctx, name, 100)
		return nil
	}
	r.notifier.Event(ctx, fmt.Sprintf("task called %s commits appears %e,and want to rollback", name, err))

	if r.policy == PolicyRetry {
		var attempts int
		backOff := r.newBackOff() // 退避算法 保证时间间隔为指数级增长
		currentInterval := 0 * time.Millisecond
		t := time.NewTimer(currentInterval)
		for {
			select {
			case <-t.C:
				shouldRetry := attempts < r.attempts
				if err = r.c.Rollback(ctx); err == nil {
					shouldRetry = false
				}
				log.FromContext(ctx).Warnf("task called %s runs for the %d time,and rollback appears %e",
					r.c.Name(), attempts, err)
				if !shouldRetry {
					t.Stop()
					return err
				}
				// 计算下一次
				currentInterval = backOff.NextBackOff()
				attempts++
				// 定时器重置
				t.Reset(currentInterval)
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			}
		}
	}
	return r.c.Rollback(ctx)
}

func (r *retryTask) newBackOff() backoff.BackOff {
	if r.attempts < 2 || r.interval <= 0 {
		return &backoff.ZeroBackOff{}
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = r.interval

	// calculate the multiplier for the given number of attempts
	// so that applying the multiplier for the given number of attempts will not exceed 2 times the initial interval
	// it allows to control the progression along the attempts
	b.Multiplier = math.Pow(2, 1/float64(r.attempts-1))

	// according to docs, b.Reset() must be called before using
	b.Reset()
	return b
}
