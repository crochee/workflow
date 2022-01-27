package taskflow

import (
	"context"
	"time"
)

type (
	Committer interface {
		Name() string
		Commit(ctx context.Context) error
		Rollback(ctx context.Context) error
	}

	// Task is library's minimum unit
	Task interface {
		Execute(ctx context.Context) error
	}

	TaskWrapper interface {
		Then(t Task) Task
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
