package workflow

import (
	"context"
	"fmt"
	"runtime"
)

type recoverTask struct {
	Task
}

func SafeTask(t Task) Task {
	return &recoverTask{
		Task: t,
	}
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
	err = rt.Task.Commit(ctx)
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
	err = rt.Task.Rollback(ctx)
	return
}
