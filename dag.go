package taskflow

import (
	"fmt"
	"github.com/crochee/taskflow/internal"
)

/// reference github.com/mostafa-asg/dag

// Dag is directed acyclic graph
type Dag struct {
	Vertexes []*Vertex
}

func (d *Dag) Compile(root *Vertex) [][]*Vertex {
	s := internal.NewStack()
	s.Push(root)
	visited := make(map[string]*Vertex)
	all := make([][]*Vertex, 0)
	for s.Length() > 0 {
		qSize := s.Length()
		tmp := make([]*Vertex, 0)
		for i := 0; i < qSize; i++ {
			//pop vertex
			currVert := s.Pop()
			if _, ok := visited[currVert.Flow.Name()]; ok {
				continue
			}
			visited[currVert.Flow.Name()] = currVert
			tmp = append(tmp, currVert)
			for _, val := range currVert.Next {
				if _, ok := visited[val.Flow.Name()]; !ok {
					s.Push(val)
				}
			}
		}
		all = append([][]*Vertex{tmp}, all...)
	}
	return nil
}

type Vertex struct {
	Flow      Flow
	Condition Condition
	Prev      []*Vertex
	Next      []*Vertex
}

func NewVertex(flow Flow) *Vertex {
	return &Vertex{
		Flow: flow,
	}
}

func (v *Vertex) AddEdge(to *Vertex) {
	v.Next = append(v.Next, to)
	to.Prev = append(to.Prev, v)
}

func (v *Vertex) AddCondition(condition Condition) {
	v.Condition = condition
}

func (v *Vertex) String() string {
	return fmt.Sprintf("name: %s - Children: %d - Value: %v\n",
		v.Flow.Name(), len(v.Next), v.Condition)
}
