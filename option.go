package taskflow

import "time"

type option struct {
	name     string
	attempt  int
	interval time.Duration
	notifier Notifier
	policy   Policy
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
