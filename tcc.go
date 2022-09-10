package workflow

import (
	"context"
)

type TCC interface {
	Info
	Try(ctx context.Context, input interface{}, callbacks ...Callback) error
	Confirm(ctx context.Context, input interface{}, callbacks ...Callback) error
	Cancel(ctx context.Context, input interface{}, callbacks ...Callback) error
}

func NewTCC(try, confirm, cancel Task, opts ...Option) TCC {
	opt := &options{
		info: DefaultTaskInfo(),
	}
	for _, o := range opts {
		o.apply(opt)
	}
	// 初始化状态
	opt.info.SetName("tcc")
	opt.info.SetState(Ready)
	opt.info.SetDescription("tcc")
	return &simpleTCC{
		Info:      opt.info,
		try:       try,
		confirm:   confirm,
		cancel:    cancel,
		callbacks: opt.callbacks,
	}
}

type simpleTCC struct {
	Info
	try       Task
	confirm   Task
	cancel    Task
	callbacks []Callback
}

func (s *simpleTCC) Try(ctx context.Context, input interface{}, callbacks ...Callback) error {
	s.SetState(Running)
	err := s.try.Execute(ctx, input)
	s.AddError(err, false)
	for _, callback := range s.callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	return err
}

func (s *simpleTCC) Confirm(ctx context.Context, input interface{}, callbacks ...Callback) error {
	err := s.confirm.Execute(ctx, input)
	s.AddError(err)
	for _, callback := range s.callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	return err
}

func (s *simpleTCC) Cancel(ctx context.Context, input interface{}, callbacks ...Callback) error {
	err := s.cancel.Execute(ctx, input)
	s.AddError(err)
	for _, callback := range s.callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	for _, callback := range callbacks {
		callback.Trigger(ctx, s.Info, input, err)
	}
	return err
}
