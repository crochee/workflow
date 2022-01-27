package taskflow

/// reference github.com/mostafa-asg/dag

// DAG is directed acyclic graph
type DAG struct {
	Vertexes []*Vertex
}

type Vertex struct {
	key, value interface{}
	Pre        []*Vertex
	Next       []*Vertex
}

func (d *DAG) AddVertex(v *Vertex) {
	d.Vertexes = append(d.Vertexes, v)
}

func (from *Vertex) AddEdge(to *Vertex)  {
	from.Next = append(from.Next, to)
	to.Pre = append(to.Pre, from)
}
