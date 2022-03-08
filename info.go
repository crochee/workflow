package workflow

import (
	"sync/atomic"
	"time"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/multierr"
)

type State string

const (
	Ready   State = "ready"
	Running State = "running"
	Success State = "success"
	Error   State = "error"
)

type Info interface {
	ID() string
	Name() string
	State() State
	Description() string
	CreateTime() time.Time
	UpdateTime() time.Time
	Error() error

	SetName(string)
	SetState(State)
	SetDescription(string)
	AddError(err error, states ...bool)
}

func DefaultTaskInfo(f ...func() time.Time) Info {
	nowFunc := time.Now
	if len(f) > 0 {
		nowFunc = f[0]
	}

	d := &defaultTaskInfo{
		id:          uuid.NewV4().String(),
		nowFunc:     nowFunc,
		name:        atomic.Value{},
		state:       atomic.Value{},
		description: atomic.Value{},
		createTime:  time.Time{},
		updateTime:  atomic.Value{},
		err:         atomic.Value{},
	}
	now := d.nowFunc()
	d.createTime = now
	d.updateTime.Store(now)
	return d
}

type defaultTaskInfo struct {
	id          string
	nowFunc     func() time.Time
	name        atomic.Value
	state       atomic.Value
	description atomic.Value
	createTime  time.Time
	updateTime  atomic.Value
	err         atomic.Value
}

func (t *defaultTaskInfo) ID() string {
	return t.id
}

func (d *defaultTaskInfo) Name() string {
	v, _ := d.name.Load().(string)
	return v
}

func (d *defaultTaskInfo) State() State {
	v, _ := d.state.Load().(State)
	return v
}

func (d *defaultTaskInfo) Description() string {
	v, _ := d.description.Load().(string)
	return v
}

func (d *defaultTaskInfo) CreateTime() time.Time {
	return d.createTime
}

func (d *defaultTaskInfo) UpdateTime() time.Time {
	v, _ := d.updateTime.Load().(time.Time)
	return v
}

func (d *defaultTaskInfo) Error() error {
	v, _ := d.err.Load().(error)
	return v
}

func (d *defaultTaskInfo) SetName(name string) {
	d.name.Store(name)
	d.updateTime.Store(d.nowFunc())
}

func (d *defaultTaskInfo) SetState(state State) {
	d.state.Store(state)
	d.updateTime.Store(d.nowFunc())
}

func (d *defaultTaskInfo) SetDescription(desc string) {
	d.description.Store(desc)
	d.updateTime.Store(d.nowFunc())
}

func (d *defaultTaskInfo) setError(err error) {
	if err != nil {
		d.err.Store(err)
		d.updateTime.Store(d.nowFunc())
	}
}

func (d *defaultTaskInfo) AddError(err error, states ...bool) {
	setState := true
	if len(states) > 0 {
		setState = states[0]
	}
	curErr := d.Error()
	if curErr != nil {
		if setState {
			d.SetState(Error)
		}
		d.setError(multierr.Append(curErr, err))
		return
	}
	if setState {
		if err != nil {
			d.SetState(Error)
			d.setError(err)
			return
		}
		d.SetState(Success)
		return
	}
	if err != nil {
		d.setError(err)
	}
}
