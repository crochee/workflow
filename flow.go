package taskflow

import (
	"github.com/crochee/lirity/id"
)

type linearFlow struct {
	name  string
	tasks []Task
	index int
}

func NewLinearFlow(opts ...Option) Flow {
	option := &options{
		name: "flow-" + id.UUID(),
	}
	for _, opt := range opts {
		opt.Apply(option)
	}
	return &linearFlow{
		name:  option.name,
		tasks: make([]Task, 0, 2),
	}
}

func (l *linearFlow) Name() string {
	return l.name
}

func (l *linearFlow) Requires(ids ...string) bool {
	// todo

	return true
}

func (l *linearFlow) Add(tasks ...Task) {
	l.tasks = append(l.tasks, tasks...)
}
