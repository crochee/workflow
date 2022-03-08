package workflow

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type retryTask struct {
	Task
	attempts int
	interval time.Duration
	policy   Policy
}

func RetryTask(t Task, opts ...RetryOption) Task {
	opt := &retryOptions{
		attempts: defaultAttempt,
		interval: defaultInterval,
		policy:   PolicyRetry,
	}
	for _, o := range opts {
		o.apply(opt)
	}
	name := t.Name()
	t.SetName(fmt.Sprintf("retry-%s", name))
	return &retryTask{
		Task:     t,
		attempts: opt.attempts,
		interval: opt.interval,
		policy:   opt.policy,
	}
}

func (rt *retryTask) Execute(ctx context.Context, input interface{}, callbacks ...Callback) error {
	err := rt.Task.Execute(ctx, input, callbacks...)
	if err == nil {
		return nil
	}
	rt.Task.SetState(Running)
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
					rt.Task.AddError(err)
					return rt.Task.Error()
				}
				if retryErr := rt.Task.Execute(ctx, input, callbacks...); retryErr == nil {
					shouldRetry = false
				} else {
					rt.Task.SetState(Running)
				}
				if !shouldRetry {
					timer.Stop()
					rt.Task.AddError(err)
					return rt.Task.Error()
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
	rt.Task.AddError(err)
	return rt.Task.Error()
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
