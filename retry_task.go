package workflow

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/multierr"
)

type Policy uint8

const (
	PolicyRetry Policy = 1 + iota
	PolicyRevert
)

type retryOption struct {
	attempts int
	interval time.Duration
	policy   Policy
}

type RetryOption func(*retryOption)

func WithAttempt(attempt int) RetryOption {
	return func(o *retryOption) {
		o.attempts = attempt
	}
}

func WithInterval(interval time.Duration) RetryOption {
	return func(o *retryOption) {
		o.interval = interval
	}
}

func WithPolicy(policy Policy) RetryOption {
	return func(o *retryOption) {
		o.policy = policy
	}
}

type retryTask struct {
	Task
	attempts int
	interval time.Duration
	policy   Policy
}

func RetryTask(t Task, opts ...RetryOption) Task {
	o := &retryOption{
		policy: PolicyRetry,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &retryTask{
		Task:     t,
		attempts: o.attempts,
		interval: o.interval,
		policy:   o.policy,
	}
}

func (rt *retryTask) Commit(ctx context.Context, opts ...TaskOption) error {
	err := rt.Task.Commit(ctx, opts...)
	if err == nil {
		return nil
	}
	if rt.policy == PolicyRetry {
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
				if retryErr := rt.Task.Commit(ctx, opts...); retryErr == nil {
					shouldRetry = false
				} else {
					err = multierr.Append(err, fmt.Errorf("%d try,%w", tempAttempts+1, retryErr))
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
