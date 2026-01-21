package topo

import (
	"iter"
	"slices"
)

type (
	EdgeFunc[T any, ID any]       func(T) iter.Seq[ID]
	IdentifierFunc[T any, ID any] func(T) ID
)

func Sort[T any, ID comparable](elements []T, elementID IdentifierFunc[T, ID], elementEdges EdgeFunc[T, ID]) error {
	sorted := make([]T, 0, len(elements))
	err := iterate(elements, elementID, elementEdges, func(_ int, _ ID, el T, _ []ID) bool {
		sorted = append(sorted, el)
		return true
	})
	copy(elements, sorted)
	return err
}

func iterate[T any, ID comparable](elements []T, elementID IdentifierFunc[T, ID], elementEdges EdgeFunc[T, ID], yield func(int, ID, T, []ID) bool) error {
	var (
		visited   = make([]bool, 2*len(elements))
		temporal  = visited[:len(elements)]
		permanent = visited[len(elements):]

		ids = make(map[ID]int, len(elements))
	)
	var visit func(ID) error
	visit = func(id ID) error {
		index := ids[id]
		if permanent[index] {
			return nil
		}
		if temporal[index] {
			return ErrCycleDetected
		}
		temporal[index] = true
		e := elements[index]
		inputs := slices.Collect(elementEdges(e))
		for _, dep := range inputs {
			if err := visit(dep); err != nil {
				return err
			}
		}
		if !yield(index, id, e, inputs) {
			return nil
		}
		permanent[index] = true
		return nil
	}
	for i, e := range elements {
		ids[elementID(e)] = i
	}
	for _, e := range elements {
		if err := visit(elementID(e)); err != nil {
			return err
		}
	}
	return nil
}
