package workflow

import (
	"context"
	"time"
)

type (
	// Task is library's minimum unit
	Task interface {
		ID() string
		Name() string
		Commit(ctx context.Context) error
		Rollback(ctx context.Context) error
	}

	Executor interface {
		Run(ctx context.Context) error
	}

	Notifier interface {
		Event(ctx context.Context, format string, v ...interface{})
		Notify(ctx context.Context, name string, progress float32)
	}

	ScheduleParser interface {
		Parse(spec string) (Scheduler, error)
	}

	Scheduler interface {
		Next(t time.Time) time.Time
	}

	Flow interface {
		ID() string
		Name() string
		Add(tasks ...Task) Flow
		ListTask() []Task
		Run(ctx context.Context) error
	}

	Condition interface {
		NextFlow() []string
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
