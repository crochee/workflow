package taskflow

import (
	"context"
	"fmt"
	"runtime"

	"github.com/crochee/lirity/log"
)

type retryTask struct {
	t      Task
	name   string
	policy Policy
}

func NewRecoverTask(t Task, opts ...Option) Task {
	o := &option{
		name:   t.Name(),
		policy: t.Policy(),
	}
	for _, opt := range opts {
		opt(o)
	}
	return &retryTask{
		t:      t,
		name:   o.name,
		policy: o.policy,
	}
}

func (r *retryTask) Name() string {
	return r.name
}

func (r *retryTask) Policy() Policy {
	return r.policy
}

func (r *retryTask) Commit(ctx context.Context) error {
	defer innerRecover(ctx)
	return r.t.Commit(ctx)
}

func (r *retryTask) Rollback(ctx context.Context) error {
	defer innerRecover(ctx)
	return r.t.Rollback(ctx)
}

func innerRecover(ctx context.Context) {
	if r := recover(); r != nil {
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
		log.FromContext(ctx).Errorf("[Recover] %e \n stack:%s", err, buf)
	}
}
