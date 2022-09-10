package workflow

import (
	"context"
)

func NewTCCPipeline(opts ...Option) *noopTCCPipeline {
	opt := &options{
		info: DefaultTaskInfo(),
	}
	for _, o := range opts {
		o.apply(opt)
	}
	// 初始化状态
	opt.info.SetName("tcc-pipeline")
	opt.info.SetState(Ready)
	opt.info.SetDescription("tcc pipeline")
	return &noopTCCPipeline{
		Info:      opt.info,
		callbacks: opt.callbacks,
	}
}

type noopTCCPipeline struct {
	Info
	callbacks []Callback
}

func (n *noopTCCPipeline) WithTCCs(tccs ...TCC) TCC {
	return &tccPipeline{
		noopTCCPipeline: n,
		tccs:            tccs,
		cur:             0,
	}
}

type tccPipeline struct {
	*noopTCCPipeline
	tccs []TCC
	cur  int
}

func (t *tccPipeline) Try(ctx context.Context, input interface{}, callbacks ...Callback) error {
	for index, tcc := range t.tccs {
		t.cur = index
		if err := tcc.Try(ctx, input); err != nil {
			t.AddError(err, false)
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

func (t *tccPipeline) Confirm(ctx context.Context, input interface{}, callbacks ...Callback) error {
	for _, tcc := range t.tccs {
		if err := tcc.Confirm(ctx, input); err != nil {
			t.AddError(err)
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

func (t *tccPipeline) Cancel(ctx context.Context, input interface{}, callbacks ...Callback) error {
	for i := t.cur; i >= 0; i-- {
		if err := t.tccs[i].Cancel(ctx, input); err != nil {
			t.AddError(err)
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
