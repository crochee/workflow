package taskflow

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	s := NewStack()
	assert.Equal(t, 0, s.Length())
	p1 := NewPipelineFlow()
	name1 := p1.Name()
	v1 := NewVertex(p1)
	s.Push(v1)
	assert.Equal(t, 1, s.Length())
	assert.Equal(t, v1, s.Top())
	p2 := NewSpawnFlow()
	name2 := p2.Name()
	v := NewVertex(p2)
	fmt.Println(v)
	s.Push(v)
	assert.Equal(t, 2, s.Length())

	assert.Equal(t, name2, s.Pop().Flow.Name())
	assert.Equal(t, name1, s.Pop().Flow.Name())
}
