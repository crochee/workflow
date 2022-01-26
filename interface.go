package taskflow

import (
	"context"
	"time"
)

type ScheduleParser interface {
	Parse(spec string) (Scheduler, error)
}

type Scheduler interface {
	Next(t time.Time) time.Time
}

type Executor interface {
	Run(ctx context.Context) error
	Stop() error
}

type Job interface {
	Do(ctx context.Context) error
	Redo(ctx context.Context) error
	Revert(ctx context.Context) error
}

type Next interface {
	NextBackOff() time.Duration
}
