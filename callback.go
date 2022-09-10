package workflow

import "context"

type Callback interface {
	Trigger(ctx context.Context, info Info, input interface{}, err error)
}
