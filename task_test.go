package workflow

import (
	"context"
	"testing"
)

func TestSimpleTask(t *testing.T) {
	f := NewFunc(UI)
	t.Log(f.ID(), f.Name())
}

func UI(ctx context.Context, input interface{}) error {
	return nil
}
