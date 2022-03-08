package workflow

import (
	"context"
	"sync"
)

func NewTCCGroup(opts ...Option) *noopTCCGroup {
	opt := &options{
		info: DefaultTaskInfo(),
	}
	for _, o := range opts {
		o.apply(opt)
	}
	// 初始化状态
	opt.info.SetName("tcc-group")
	opt.info.SetState(Ready)
	opt.info.SetDescription("tcc group")
	return &noopTCCGroup{
		Info:      opt.info,
		callbacks: opt.callbacks,
	}
}

type noopTCCGroup struct {
	Info
	callbacks []Callback
}

func (n *noopTCCGroup) WithTCCs(tccs ...TCC) TCC {
	markedTccs := make([]*markedTCC, 0, len(tccs))
	for _, tcc := range tccs {
		markedTccs = append(markedTccs, &markedTCC{
			TCC:    tcc,
			marked: false,
		})
	}
	return &tccGroup{
		noopTCCGroup: n,
		tccs:         markedTccs,
		errOnce:      sync.Once{},
	}
}

type markedTCC struct {
	TCC
	marked bool
}

type tccGroup struct {
	*noopTCCGroup
	tccs    []*markedTCC
	errOnce sync.Once
}

func (t *tccGroup) doTry(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup,
	index int, task *markedTCC, input interface{}) {
	defer wg.Done()
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		err = task.Try(ctx, input)
		if err != nil {
			t.errOnce.Do(cancel)
		}
		// 标记已经执行的
		task.marked = true
	}
	t.AddError(err, false)
}

func (t *tccGroup) Try(ctx context.Context, input interface{}, callbacks ...Callback) error {
	t.SetState(Running)
	newCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for index, tcc := range t.tccs {
		wg.Add(1)
		go t.doTry(newCtx, cancel, &wg, index, tcc, input)
	}
	wg.Wait()
	t.errOnce.Do(cancel)

	err := t.Error()
	for _, callback := range t.callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	return err
}

func (t *tccGroup) doConfirm(ctx context.Context, wg *sync.WaitGroup,
	task *markedTCC, input interface{}) {
	err := task.Confirm(ctx, input)
	t.AddError(err)
	wg.Done()
}

func (t *tccGroup) Confirm(ctx context.Context, input interface{}, callbacks ...Callback) error {
	var wg sync.WaitGroup
	for _, tcc := range t.tccs {
		if !tcc.marked {
			continue
		}
		wg.Add(1)
		go t.doConfirm(ctx, &wg, tcc, input)
	}
	wg.Wait()
	err := t.Error()
	for _, callback := range t.callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	return err
}

func (t *tccGroup) doCancel(ctx context.Context, wg *sync.WaitGroup,
	task *markedTCC, input interface{}) {
	err := task.Confirm(ctx, input)
	t.AddError(err)
	wg.Done()
}

func (t *tccGroup) Cancel(ctx context.Context, input interface{}, callbacks ...Callback) error {
	var wg sync.WaitGroup
	for _, tcc := range t.tccs {
		if !tcc.marked {
			continue
		}
		wg.Add(1)
		go t.doCancel(ctx, &wg, tcc, input)
	}
	wg.Wait()
	err := t.Error()
	for _, callback := range t.callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, t.Info, input, err)
	}
	return err
}
