package taskflow

/// reference github.com/mostafa-asg/dag

// DAG is directed acyclic graph
type DAG struct {
	Vertexes []*Vertex
}

func (d *DAG) AddVertex(v *Vertex) {
	d.Vertexes = append(d.Vertexes, v)
}

type Vertex struct {
	flow      Flow
	condition Condition
	Pre       []*Vertex
	Next      []*Vertex
}

func NewVertex(flow Flow) *Vertex {
	return &Vertex{
		flow: flow,
	}
}

func (from *Vertex) AddEdge(condition Condition, to *Vertex) {
	from.condition = condition
	from.Next = append(from.Next, to)
	to.Pre = append(to.Pre, from)
}
