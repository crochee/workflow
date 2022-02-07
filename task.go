package taskflow

import (
	"context"
	"fmt"
	"runtime"
)

type recoverTask struct {
	t Task
}

func SafeTask(t Task) Task {
	return &recoverTask{
		t: t,
	}
}

func (rt *recoverTask) ID() string {
	return rt.t.ID()
}

func (rt *recoverTask) Name() string {
	return rt.t.Name()
}

func (rt *recoverTask) Policy() Policy {
	return rt.t.Policy()
}

func (rt *recoverTask) Commit(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("found:%v,trace:%s", r, buf)
			}
		}
	}()
	err = rt.t.Commit(ctx)
	return
}

func (rt *recoverTask) Rollback(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			var ok bool
			if err, ok = r.(error); !ok {
				err = fmt.Errorf("found:%v,trace:%s", r, buf)
			}
		}
	}()
	err = rt.t.Rollback(ctx)
	return
}
