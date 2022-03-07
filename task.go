package workflow

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	"go.uber.org/multierr"
)

type State string

const (
	Ready   State = "ready"
	Running State = "running"
	Success State = "success"
	Error   State = "error"
)

type taskOption struct {
	name        string
	description string
	meta        map[string]interface{}
	callbacks   []Callback
	nowFunc     func() time.Time
	tasks       []Task
}

type TaskOption func(*taskOption)

func WithName(name string) TaskOption {
	return func(option *taskOption) {
		option.name = name
	}
}

func WithDescription(description string) TaskOption {
	return func(option *taskOption) {
		option.description = description
	}
}

func WithMeta(meta map[string]interface{}) TaskOption {
	return func(option *taskOption) {
		option.meta = meta
	}
}

func WithCallbacks(callbacks ...Callback) TaskOption {
	return func(option *taskOption) {
		option.callbacks = callbacks
	}
}

func WithNowFunc(nowFunc func() time.Time) TaskOption {
	return func(option *taskOption) {
		option.nowFunc = nowFunc
	}
}

func WithTasks(tasks ...Task) TaskOption {
	return func(option *taskOption) {
		option.tasks = tasks
	}
}

// Task is library's minimum unit
type Task interface {
	ID() string
	Name() string
	State() (State, error)
	Description() string
	CreateTime() time.Time
	UpdateTime() time.Time
	Meta() map[string]interface{}

	Commit(ctx context.Context, opts ...TaskOption) error
	Rollback(ctx context.Context, opts ...TaskOption) error
}

func NewFuncTask(f func(context.Context) error, opts ...TaskOption) Task {
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	o := &taskOption{
		name:        funcName,
		description: "it's a simple function task",
		meta:        map[string]interface{}{},
		callbacks:   make([]Callback, 0),
		nowFunc:     time.Now,
	}
	for _, opt := range opts {
		opt(o)
	}

	h := md5.New()
	_, _ = fmt.Fprint(h, funcName)
	id := hex.EncodeToString(h.Sum(nil))
	now := o.nowFunc()
	return &funcTask{
		id:          id,
		name:        o.name,
		state:       Ready,
		description: o.description,
		createTime:  now,
		updateTime:  now,
		meta:        o.meta,
		callbacks:   o.callbacks,
		now:         o.nowFunc,
		f:           f,
		err:         nil,
		rwMutex:     sync.RWMutex{},
	}
}

type funcTask struct {
	id          string
	name        string
	state       State
	description string
	createTime  time.Time
	updateTime  time.Time
	meta        map[string]interface{}
	callbacks   []Callback
	now         func() time.Time

	f       func(context.Context) error
	err     error
	rwMutex sync.RWMutex
}

func (f *funcTask) ID() string {
	return f.id
}

func (f *funcTask) Name() string {
	return f.name
}

func (f *funcTask) State() (State, error) {
	f.rwMutex.RLock()
	defer f.rwMutex.RUnlock()
	return f.state, f.err
}

func (f *funcTask) Description() string {
	return f.description
}

func (f *funcTask) CreateTime() time.Time {
	return f.createTime
}

func (f *funcTask) UpdateTime() time.Time {
	return f.createTime
}

func (f *funcTask) Meta() map[string]interface{} {
	return f.meta
}

func (f *funcTask) Commit(ctx context.Context, opts ...TaskOption) error {
	o := &taskOption{}
	for _, opt := range opts {
		opt(o)
	}
	callbacks := append(f.callbacks, o.callbacks...)

	f.rwMutex.Lock()
	f.state = Running
	f.updateTime = f.now()
	f.rwMutex.Unlock()

	for _, callback := range callbacks {
		callback.Trigger(ctx, f)
	}

	err := f.f(ctx)

	f.rwMutex.Lock()
	if err != nil {
		f.err = err
		f.state = Error
	} else {
		f.state = Success
	}
	f.updateTime = f.now()
	f.rwMutex.Unlock()

	for _, callback := range callbacks {
		callback.Trigger(ctx, f)
	}
	return err
}

func (f *funcTask) Rollback(context.Context, ...TaskOption) error {
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

func (rt *recoverTask) Commit(ctx context.Context, opts ...TaskOption) (err error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			err = multierr.Append(err, fmt.Errorf("[Recover] found:%v,trace:\n%s", r, buf))
		}
	}()
	err = rt.Task.Commit(ctx, opts...)
	return
}

func (rt *recoverTask) Rollback(ctx context.Context, opts ...TaskOption) (err error) {
	defer func() {
		if r := recover(); r != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			err = multierr.Append(err, fmt.Errorf("[Recover] found:%v,trace:\n%s", r, buf))
		}
	}()
	err = rt.Task.Rollback(ctx, opts...)
	return
}
