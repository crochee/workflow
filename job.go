package taskflow

import "context"

type Job struct {
	RetryHandler
	notifier Notifier
}

func (j *Job) UpdateProgress(ctx context.Context, progress float32) error {
	return j.notifier.Notify(ctx, progress)
}
