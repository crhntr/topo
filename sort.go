package topological

import (
	"cmp"
	"fmt"
	"slices"
)

func Sort[T any, ID cmp.Ordered](elements []T, elementID func(T) ID, elementEdges func(T) []ID) ([]T, error) {
	var (
		visited   = make([]bool, 2*len(elements))
		temporal  = visited[:len(elements)]
		permanent = visited[len(elements):]

		ids    = make(map[ID]int, len(elements))
		sorted = make([]T, 0, len(elements))
	)
	var visit func(ID) error
	visit = func(id ID) error {
		if permanent[ids[id]] {
			return nil
		}
		if temporal[ids[id]] {
			return fmt.Errorf("cycle detected")
		}
		temporal[ids[id]] = true
		e := elements[ids[id]]
		for _, dep := range elementEdges(e) {
			if err := visit(dep); err != nil {
				return err
			}
		}
		sorted = append(sorted, e)
		permanent[ids[id]] = true
		return nil
	}
	slices.SortFunc(elements, func(a, b T) int {
		return cmp.Compare(elementID(a), elementID(b))
	})
	for i, e := range elements {
		ids[elementID(e)] = i
	}
	for _, e := range elements {
		if err := visit(elementID(e)); err != nil {
			return nil, err
		}
	}
	return sorted, nil
}
