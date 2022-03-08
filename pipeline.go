package workflow

import "context"

func NewTaskPipeline(opts ...Option) *noopTaskPipeline {
	opt := &options{
		info: DefaultTaskInfo(),
	}
	for _, o := range opts {
		o.apply(opt)
	}
	// 初始化状态
	opt.info.SetName("task-pipeline")
	opt.info.SetState(Ready)
	opt.info.SetDescription("task pipeline")
	return &noopTaskPipeline{
		Info:      opt.info,
		callbacks: opt.callbacks,
	}
}

type noopTaskPipeline struct {
	Info
	callbacks []Callback
}

func (p *noopTaskPipeline) WithTasks(tasks ...Task) Task {
	return &taskPipeline{
		noopTaskPipeline: p,
		tasks:            tasks,
	}
}

type taskPipeline struct {
	*noopTaskPipeline
	tasks []Task
}

func (t *taskPipeline) Execute(ctx context.Context, input interface{}, callbacks ...Callback) error {
	for _, tempTask := range t.tasks {
		if err := tempTask.Execute(ctx, input); err != nil {
			t.AddError(err)
			break
		}
	}
	err := t.Error()
	for _, callback := range t.callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	return err
}
