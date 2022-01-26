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

	// Task is library's minimum unit
	Task interface {
		ID() uint64
		Name() string
		PreExecute(ctx context.Context) error
		Execute(ctx context.Context) error
		AfterExecute(ctx context.Context) error
		PreRollback(ctx context.Context) error
		Rollback(ctx context.Context) error
		AfterRollback(ctx context.Context) error
	}

	Notifier interface {
		Notify(ctx context.Context, progress float32) error
	}

	RetryHandler interface {
		Task
		Policy() Policy
		OnFailure(ctx context.Context, policy Policy, attempts int, interval time.Duration) error
	}

	Flow interface {
		ID() uint64
		Name() string
		Requires(ids ...uint64) bool
		Add(flows ...Flow)
		IterLinks() []*FlowRelation
		IterJobs() []*JobRelation
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
		A  Flow
		B Flow
		Meta map[string]interface{}
	}

	JobRelation struct {
		A  Job
		B Job
		Meta map[string]interface{}
	}
)

type Policy uint8

const (
	Revert Policy = 1 + iota
	RevertAll
	Retry
)
