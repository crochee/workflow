package taskflow

type linearFlow struct {
	id       uint64
	name     string
	listFlow []Flow
	cur      int
}

func (l *linearFlow) ID() uint64 {
	return l.id
}

func (l *linearFlow) Name() string {
	return l.name
}

func (l *linearFlow) Requires(ids ...uint64) bool {
	// todo

	return true
}

func (l *linearFlow) Add(flows ...Flow) {
	l.listFlow = append(l.listFlow, flows...)
}

func (l *linearFlow) IterLinks() []*FlowRelation {
	length := len(l.listFlow)
	result := make([]*FlowRelation, 0, length)
	for index, v := range l.listFlow {
		var value *FlowRelation
		if index < length-1 {
			value = &FlowRelation{
				A:    v,
				B:    l.listFlow[index+1],
				Meta: nil,
			}
		}
		result = append(result, value)
	}
	return result
}

func (l *linearFlow) IterJobs() []*JobRelation {
	panic("implement me")
}
