package taskflow

import (
	"context"
	"fmt"
	"runtime"

	"github.com/crochee/lirity/log"
)

type chain struct {
	wrappers []TaskWrapper
}

func NewChain(c ...TaskWrapper) TaskWrapper {
	return chain{c}
}

func (c chain) Then(t Task) Task {
	for i := range c.wrappers {
		t = c.wrappers[len(c.wrappers)-i-1].Then(t)
	}
	return t
}

type FuncTaskWrapper func(Task) Task

func (f FuncTaskWrapper) Then(t Task) Task {
	return f(t)
}

// Recover panics in wrapped Tasks and log them with the provided logger.
func Recover() TaskWrapper {
	return FuncTaskWrapper(func(task Task) Task {
		return FuncTask(func(ctx context.Context) error {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					log.FromContext(ctx).Errorf("[Recover] %e \n stack:%s", err, buf)
				}
			}()
			return task.Execute(ctx)
		})
	})
}
