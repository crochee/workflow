package internal

import (
	"fmt"
	"github.com/crochee/taskflow"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	s := NewStack()
	assert.Equal(t, 0, s.length)
	p1 := taskflow.NewPipelineFlow()
	name1 := p1.Name()
	s.Push(taskflow.NewVertex(p1))
	assert.Equal(t, 1, s.length)

	p2 := taskflow.NewSpawnFlow()
	name2 := p2.Name()
	v := taskflow.NewVertex(p2)
	fmt.Println(v)
	s.Push(v)
	assert.Equal(t, 2, s.length)

	assert.Equal(t, name2, s.Pop().Flow.Name())
	assert.Equal(t, name1, s.Pop().Flow.Name())
}
