package topo

import (
	"fmt"
)

type (
	EdgeFunc[T any, ID any]       func(T) []ID
	IdentifierFunc[T any, ID any] func(T) ID
)

func Sort[T any, ID comparable](elements []T, elementID IdentifierFunc[T, ID], elementEdges EdgeFunc[T, ID]) error {
	var (
		visited   = make([]bool, 2*len(elements))
		temporal  = visited[:len(elements)]
		permanent = visited[len(elements):]

		ids    = make(map[ID]int, len(elements))
		sorted = make([]T, 0, len(elements))
	)
	var visit func(ID) error
	visit = func(id ID) error {
		index := ids[id]
		if permanent[index] {
			return nil
		}
		if temporal[index] {
			return fmt.Errorf("cycle detected")
		}
		temporal[index] = true
		e := elements[index]
		for _, dep := range elementEdges(e) {
			if err := visit(dep); err != nil {
				return err
			}
		}
		sorted = append(sorted, e)
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
	copy(elements, sorted)
	return nil
}
