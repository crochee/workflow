package workflow

import "context"

type Callback interface {
	Trigger(ctx context.Context, task Task)
}
