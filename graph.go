package topo

type graphNode[T any, ID comparable, Data any] struct {
	index    int
	id       ID
	element  T
	parents  []*graphNode[T, ID, Data]
	children []*graphNode[T, ID, Data]
	data     Data
}

func newGraph[T any, ID comparable, Data any](elements []T, elementID IdentifierFunc[T, ID], elementEdges EdgeFunc[T, ID], initData func(*graphNode[T, ID, Data])) ([]*graphNode[T, ID, Data], error) {
	var roots []*graphNode[T, ID, Data]
	m := make(map[ID]*graphNode[T, ID, Data])
	err := iterate(elements, elementID, elementEdges, func(index int, id ID, el T, requirements []ID) bool {
		n := &graphNode[T, ID, Data]{
			index:    index,
			id:       id,
			element:  el,
			parents:  nil,
			children: nil,
		}
		if _, ok := m[id]; ok {
			panic("element must not be visited more than once")
		}
		m[id] = n
		if len(requirements) == 0 {
			if initData != nil {
				initData(n)
			}
			roots = append(roots, n)
			return true
		}

		n.parents = make([]*graphNode[T, ID, Data], 0, len(requirements))
		for _, req := range requirements {
			up, ok := m[req]
			if !ok {
				panic("requirements must be visited first")
			}
			up.children = append(up.children, n)
			n.parents = append(n.parents, up)
		}
		if initData != nil {
			initData(n)
		}
		return true
	})
	m = nil
	return roots, err
}
