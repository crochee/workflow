package workflow

import "time"

type options struct {
	info      Info
	callbacks []Callback
}

type Option interface {
	apply(*options)
}

func WithCallbacks(callbacks ...Callback) Option {
	return &callbacksOption{callbacks}
}

type callbacksOption struct {
	callbacks []Callback
}

func (c callbacksOption) apply(opts *options) {
	opts.callbacks = c.callbacks
}

func WithInfo(info Info) Option {
	return infoOption{info}
}

type infoOption struct {
	info Info
}

func (i infoOption) apply(opts *options) {
	opts.info = i.info
}

type Policy uint8

const (
	PolicyRetry Policy = 1 + iota
	PolicyRevert

	defaultAttempt  = 30
	defaultInterval = 60 * time.Second
)

type retryOptions struct {
	attempts int
	interval time.Duration
	policy   Policy
}

type RetryOption interface {
	apply(*retryOptions)
}

type attemptsRetryOptions struct {
	attempts int
}

func (a attemptsRetryOptions) apply(opts *retryOptions) {
	opts.attempts = a.attempts
}

func WithAttempt(attempt int) RetryOption {
	return attemptsRetryOptions{attempts: attempt}
}

type intervalRetryOptions struct {
	interval time.Duration
}

func (i intervalRetryOptions) apply(opts *retryOptions) {
	opts.interval = i.interval
}

func WithInterval(interval time.Duration) RetryOption {
	return intervalRetryOptions{interval}
}

type policyRetryOptions struct {
	policy Policy
}

func (p policyRetryOptions) apply(opts *retryOptions) {
	opts.policy = p.policy
}

func WithPolicy(policy Policy) RetryOption {
	return policyRetryOptions{policy}
}
