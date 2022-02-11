package workflow

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"

	"go.uber.org/multierr"
)

// Task is library's minimum unit
type Task interface {
	ID() string
	Name() string
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

func NewFuncTask(f func(context.Context) error) Task {
	return funcTask(f)
}

type funcTask func(context.Context) error

func (f funcTask) ID() string {
	h := md5.New()
	_, _ = fmt.Fprint(h, f.Name())
	return hex.EncodeToString(h.Sum(nil))
}

func (f funcTask) Name() string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func (f funcTask) Commit(ctx context.Context) error {
	return f(ctx)
}

func (f funcTask) Rollback(context.Context) error {
	return nil
}

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
			err = multierr.Append(err, fmt.Errorf("[Recover] found:%v,trace:\n%s", r, buf))
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
			err = multierr.Append(err, fmt.Errorf("[Recover] found:%v,trace:\n%s", r, buf))
		}
	}()
	err = rt.Task.Rollback(ctx)
	return
}
