package taskflow

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	s := NewStack()
	assert.Equal(t, 0, s.length)
	p1 := NewPipelineFlow()
	name1 := p1.Name()
	s.Push(NewVertex(p1))
	assert.Equal(t, 1, s.length)

	p2 := NewSpawnFlow()
	name2 := p2.Name()
	v := NewVertex(p2)
	fmt.Println(v)
	s.Push(v)
	assert.Equal(t, 2, s.length)

	assert.Equal(t, name2, s.Pop().Flow.Name())
	assert.Equal(t, name1, s.Pop().Flow.Name())
}
