package workflow

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"runtime"
	"sync"

	"go.uber.org/multierr"
)

type State string

const (
	Ready   State = "ready"
	Running State = "running"
	Success State = "success"
	Error   State = "error"
)

type option struct {
	name        string
	description string
	meta        map[string]interface{}
	callbacks   []Callback
	tasks       []Task
}

type Option func(*option)

func WithName(name string) Option {
	return func(option *option) {
		option.name = name
	}
}

func WithDescription(description string) Option {
	return func(option *option) {
		option.description = description
	}
}

func WithMeta(meta map[string]interface{}) Option {
	return func(option *option) {
		option.meta = meta
	}
}

func WithCallbacks(callbacks ...Callback) Option {
	return func(option *option) {
		option.callbacks = callbacks
	}
}

func WithTasks(tasks ...Task) Option {
	return func(option *option) {
		option.tasks = tasks
	}
}

// Task is library's minimum unit
type Task interface {
	ID() string
	Name() string
	State() (State, error)
	Description() string
	Meta() map[string]interface{}

	Commit(ctx context.Context, opts ...Option) error
	Rollback(ctx context.Context, opts ...Option) error
}

func NewFuncTask(f func(context.Context) error, opts ...Option) Task {
	funcName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	o := &option{
		name:        funcName,
		description: "it's a simple function task",
		meta:        map[string]interface{}{},
		callbacks:   make([]Callback, 0),
	}
	for _, opt := range opts {
		opt(o)
	}

	h := md5.New()
	_, _ = fmt.Fprint(h, funcName)
	id := hex.EncodeToString(h.Sum(nil))
	return &funcTask{
		id:          id,
		name:        o.name,
		state:       Ready,
		description: o.description,
		meta:        o.meta,
		callbacks:   o.callbacks,
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
	meta        map[string]interface{}
	callbacks   []Callback

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

func (f *funcTask) Meta() map[string]interface{} {
	return f.meta
}

func (f *funcTask) Commit(ctx context.Context, opts ...Option) error {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}
	callbacks := append(f.callbacks, o.callbacks...)

	f.rwMutex.Lock()
	f.state = Running
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
	f.rwMutex.Unlock()

	for _, callback := range callbacks {
		callback.Trigger(ctx, f)
	}
	return err
}

func (f *funcTask) Rollback(context.Context, ...Option) error {
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

func (rt *recoverTask) Commit(ctx context.Context, opts ...Option) (err error) {
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

func (rt *recoverTask) Rollback(ctx context.Context, opts ...Option) (err error) {
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
