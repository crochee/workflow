package taskflow

import (
	"context"
	"time"
)

type option struct {
	name     string
	attempt  int
	interval time.Duration
	notifier Notifier
	policy   Policy
	recover  func(ctx context.Context, notifier Notifier)
}

type Option func(*option)

func WithName(name string) Option {
	return func(o *option) {
		o.name = name
	}
}

func WithAttempt(attempt int) Option {
	return func(o *option) {
		o.attempt = attempt
	}
}

func WithInterval(interval time.Duration) Option {
	return func(o *option) {
		o.interval = interval
	}
}

func WithNotifier(notifier Notifier) Option {
	return func(o *option) {
		o.notifier = notifier
	}
}

func WithPolicy(policy Policy) Option {
	return func(o *option) {
		o.policy = policy
	}
}

func WithRecover(recover func(context.Context, Notifier)) Option {
	return func(o *option) {
		o.recover = recover
	}
}
