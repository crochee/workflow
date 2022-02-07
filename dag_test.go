package workflow

import "testing"

func TestVertex_String(t *testing.T) {
	v := NewVertex(NewPipelineFlow())
	t.Log(v)
}
