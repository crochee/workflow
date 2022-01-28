package taskflow

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/crochee/lirity/log"
)

type FuncTask func(ctx context.Context) error

func (f FuncTask) Execute(ctx context.Context) error { return f(ctx) }

type commitRetryTask struct {
	c        Committer
	notifier Notifier
	policy   Policy
	attempts int
	interval time.Duration
}

func NewTask(c Committer, opts ...Option) Task {
	o := &option{
		notifier: NoneNotify{},
		policy:   PolicyRevert,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &commitRetryTask{
		c:        c,
		policy:   o.policy,
		attempts: o.attempt,
		interval: o.interval,
		notifier: o.notifier,
	}
}

func (r *commitRetryTask) Execute(ctx context.Context) error {
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
		var (
			attempts  int
			breakFlag bool
		)
		backOff := r.newBackOff() // 退避算法 保证时间间隔为指数级增长
		currentInterval := 0 * time.Millisecond
		t := time.NewTimer(currentInterval)
		for {
			select {
			case <-t.C:
				shouldRetry := attempts < r.attempts
				if err = r.c.Commit(ctx); err == nil {
					shouldRetry = false
				}
				log.FromContext(ctx).Warnf("task called %s runs for the %d time,and commit appears %e",
					r.c.Name(), attempts, err)
				if !shouldRetry {
					t.Stop()
					breakFlag = true
					break
				}
				// 计算下一次
				currentInterval = backOff.NextBackOff()
				attempts++
				// 定时器重置
				t.Reset(currentInterval)
			case <-ctx.Done():
				t.Stop()
				err = ctx.Err()
				breakFlag = true
			}
			if breakFlag {
				break
			}
		}
	}
	if err == nil {
		r.notifier.Event(ctx, fmt.Sprintf("task called %s ends running", name))
		r.notifier.Notify(ctx, name, 100)
		return nil
	}
	r.notifier.Event(ctx, fmt.Sprintf("task called %s commits appears %e,and want to rollback", name, err))

	err = r.c.Rollback(ctx)
	r.notifier.Event(ctx, fmt.Sprintf("task called %s rollbacks appears %e", name, err))
	r.notifier.Notify(ctx, name, 0)
	return err
}

func (r *commitRetryTask) newBackOff() backoff.BackOff {
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
