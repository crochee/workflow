package workflow

import "context"

type Notifier interface {
	Event(ctx context.Context, format string, v ...interface{})
	Notify(ctx context.Context, name string, progress float32)
}

type NoneNotify struct {
}

func (n NoneNotify) Event(ctx context.Context, format string, v ...interface{}) {

}

func (n NoneNotify) Notify(ctx context.Context, name string, progress float32) {
}
