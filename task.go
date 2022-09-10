package workflow

import (
	"context"
	"fmt"
	"reflect"
	"runtime"

	"go.uber.org/multierr"
)

// Task is library's minimum unit
type Task interface {
	Info
	Execute(ctx context.Context, input interface{}, callbacks ...Callback) error
}

func NewFunc(f func(context.Context, interface{}) error, opts ...Option) Task {
	opt := &options{
		info: DefaultTaskInfo(),
	}
	for _, o := range opts {
		o.apply(opt)
	}
	// 初始化状态
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	opt.info.SetName(funcName)
	opt.info.SetState(Ready)
	opt.info.SetDescription("task")
	return &funcTask{
		Info:      opt.info,
		f:         f,
		callbacks: opt.callbacks,
	}
}

type funcTask struct {
	Info
	f         func(context.Context, interface{}) error
	callbacks []Callback
}

func (f *funcTask) Execute(ctx context.Context, input interface{}, callbacks ...Callback) (err error) {
	f.SetState(Running)
	panicked := true
	defer func() {
		if panicked {
			if r := recover(); r != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				err = multierr.Append(err, fmt.Errorf("[Recover] found:%v,trace:\n%s", r, buf))
			}
		}
		f.AddError(err)

		for _, callback := range f.callbacks {
			callback.Trigger(ctx, f.Info, input, err)
		}
		for _, callback := range callbacks {
			callback.Trigger(ctx, f.Info, input, err)
		}
	}()
	err = f.f(ctx, input)
	panicked = false
	return
}

func NewTCCTask(t TCC, opts ...Option) *simpleTCCTask {
	opt := &options{
		info: DefaultTaskInfo(),
	}
	for _, o := range opts {
		o.apply(opt)
	}
	// 初始化状态
	opt.info.SetName(fmt.Sprintf("%s-task", t.Name()))
	opt.info.SetState(Ready)
	opt.info.SetDescription("tcc task")
	return &simpleTCCTask{
		Info:      opt.info,
		tcc:       t,
		callbacks: opt.callbacks,
	}
}

type simpleTCCTask struct {
	Info
	tcc       TCC
	callbacks []Callback
}

func (t *simpleTCCTask) Strict() Task {
	return &strictTask{simpleTCCTask: t}
}

func (t *simpleTCCTask) Inert() Task {
	return &inertTask{simpleTCCTask: t}
}

type strictTask struct {
	*simpleTCCTask
}

func (s *strictTask) Execute(ctx context.Context, input interface{}, callbacks ...Callback) error {
	s.SetState(Running)
	err := s.tcc.Try(ctx, input)
	if err == nil {
		err = multierr.Append(err, s.tcc.Confirm(ctx, input))
	} else {
		err = multierr.Append(err, s.tcc.Cancel(ctx, input))
	}
	s.AddError(err)

	for _, callback := range s.callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}

	return err
}

type inertTask struct {
	*simpleTCCTask
}

func (s *inertTask) Execute(ctx context.Context, input interface{}, callbacks ...Callback) error {
	s.SetState(Running)
	err := s.tcc.Try(ctx, input)
	if err == nil {
		err = s.tcc.Confirm(ctx, input)
	} else {
		s.AddError(err, false)
		err = s.tcc.Cancel(ctx, input)
	}

	s.AddError(err)

	for _, callback := range s.callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	return err
}
