package workflow

type element struct {
	next  *element
	value *Vertex
}

type stack struct {
	top    *element
	length int
}

func NewStack() *stack {
	return &stack{}
}

func (s *stack) Push(vertex *Vertex) {
	entry := &element{next: s.top, value: vertex}
	s.top = entry
	s.length++
}

func (s *stack) Pop() *Vertex {
	if s.length == 0 {
		return nil
	}
	entry := s.top
	s.top = entry.next
	s.length--
	return entry.value
}

func (s *stack) Top() *Vertex {
	if s.length == 0 {
		return nil
	}
	return s.top.value
}

func (s *stack) Length() int {
	return s.length
}
