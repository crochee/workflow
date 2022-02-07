package taskflow

import "testing"

func TestVertex_String(t *testing.T) {
	v := NewVertex(NewPipelineFlow())
	t.Log(v)
}
