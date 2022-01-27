package taskflow

import (
	"context"
	"time"
)

type (
	ScheduleParser interface {
		Parse(spec string) (Scheduler, error)
	}

	Scheduler interface {
		Next(t time.Time) time.Time
	}

	// Atom is library's minimum unit
	Atom interface {
		Name() string
		PreExecute(ctx context.Context) error
		Execute(ctx context.Context) error
		AfterExecute(ctx context.Context) error
		PreRollback(ctx context.Context) error
		Rollback(ctx context.Context) error
		AfterRollback(ctx context.Context) error
	}

	Task interface {
		Scheduler
		Atom
		Notifier
	}

	Notifier interface {
		Notify(ctx context.Context, progress float32) error
	}

	Retry interface {
		Scheduler
		Atom
		Policy() Policy
		OnFailure(ctx context.Context, policy Policy, attempts int, interval time.Duration) error
	}

	Flow interface {
		Name() string
		Requires(names ...string) bool
		Add(tasks ...Task)
	}

	Storage interface {
	}

	Engine interface {
		Notifier
		Storage
		Statistics() map[string]interface{}
		Reset()
		Prepare()
		Run(ctx context.Context) error
		Suspend()
	}
)

type (
	FlowRelation struct {
		A    Flow
		B    Flow
		Meta map[string]interface{}
	}

	JobRelation struct {
		A    Task
		B    Task
		Meta map[string]interface{}
	}
)

type Policy uint8

const (
	PolicyRevert Policy = 1 + iota
	PolicyRevertAll
	PolicyRetry
)
