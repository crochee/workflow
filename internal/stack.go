package internal

import "github.com/crochee/taskflow"

type element struct {
	next  *element
	value *taskflow.Vertex
}

type stack struct {
	top    *element
	length int
}

func NewStack() *stack {
	return &stack{}
}

func (s *stack) Push(vertex *taskflow.Vertex) {
	entry := &element{next: s.top, value: vertex}
	s.top = entry
	s.length++
}

func (s *stack) Pop() *taskflow.Vertex {
	if s.length == 0 {
		return nil
	}
	entry := s.top
	s.top = entry.next
	s.length--
	return entry.value
}

func (s *stack) Top() *taskflow.Vertex {
	if s.length == 0 {
		return nil
	}
	return s.top.value
}

func (s *stack) Length() int {
	return s.length
}
