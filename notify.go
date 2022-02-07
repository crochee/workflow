package workflow

import "context"

type NoneNotify struct {
}

func (n NoneNotify) Event(ctx context.Context, format string, v ...interface{}) {

}

func (n NoneNotify) Notify(ctx context.Context, name string, progress float32) {
}
