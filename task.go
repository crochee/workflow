package taskflow

import (
	"context"
	"fmt"
	"runtime"
)

type recoverTask struct {
	t        Task
	id       string
	name     string
	policy   Policy
	notifier Notifier
	recover  func(ctx context.Context, notifier Notifier)
}

func NewRecoverTask(t Task, opts ...Option) Task {
	o := &option{
		name:     t.Name(),
		notifier: NoneNotify{},
		policy:   t.Policy(),
		recover:  innerRecover,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &recoverTask{
		t:        t,
		id:       t.ID(),
		name:     o.name,
		policy:   o.policy,
		notifier: o.notifier,
		recover:  o.recover,
	}
}

func (r *recoverTask) ID() string {
	return r.id
}

func (r *recoverTask) Name() string {
	return r.name
}

func (r *recoverTask) Policy() Policy {
	return r.policy
}

func (r *recoverTask) Commit(ctx context.Context) error {
	defer r.recover(ctx, r.notifier)
	return r.t.Commit(ctx)
}

func (r *recoverTask) Rollback(ctx context.Context) error {
	defer r.recover(ctx, r.notifier)
	return r.t.Rollback(ctx)
}

func innerRecover(ctx context.Context, notifier Notifier) {
	if r := recover(); r != nil {
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
		notifier.Event(ctx, "[Recover] %e \n stack:%s", err, buf)
	}
}
