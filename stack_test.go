package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	s := NewStack()
	assert.Equal(t, 0, s.Length())
	v1 := 1
	s.Push(v1)
	assert.Equal(t, 1, s.Length())
	assert.Equal(t, v1, s.Top())

	v2 := 2
	s.Push(v2)
	assert.Equal(t, 2, s.Length())

	assert.Equal(t, v2, s.Pop())
	assert.Equal(t, v1, s.Pop())
	assert.Equal(t, 0, s.Length())
}
