package topological

import (
	"cmp"
	"context"
	"slices"
	"sync"
)

type Result[V any] struct {
	Value V
	Err   error
}

type TaskFunc[T, V any] func(T, context.Context, []V) (V, error)

type result[ID cmp.Ordered, V any] struct {
	ID    ID
	Value V
	Error error
}

func Tasks[ID cmp.Ordered, T, V any](ctx context.Context, elements []T, elementID IdentifierFunc[T, ID], elementEdges EdgeFunc[T, ID], elementTask TaskFunc[T, V]) error {
	err := Sort[T, ID](elements, elementID, elementEdges)
	if err != nil {
		return err
	}
	results := make([]V, len(elements))
	done := make([]ID, 0, len(elements))

	consumeResult := func(r result[ID, V]) error {
		if r.Error != nil {
			return r.Error
		}
		for i := range elements {
			if elementID(elements[i]) == r.ID {
				done = append(done, r.ID)
				results[i] = r.Value
				break
			}
		}
		return nil
	}
	inputs := func(index int, in []ID) []V {
		params := make([]V, len(in))
		for i, id := range in {
			for j := range elements[:index] {
				if elementID(elements[j]) == id {
					params[i] = results[j]
					break
				}
			}
		}
		return params
	}

	c := make(chan result[ID, V])

	wg := sync.WaitGroup{}
	var cleanup sync.Once

	next := 0
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r, ok := <-c:
			if !ok {
				break loop
			}
			if err := consumeResult(r); err != nil {
				return err
			}
		default:
			if next >= len(elements) {
				cleanup.Do(func() {
					go func() {
						wg.Wait()
						close(c)
					}()
				})
				continue
			}
			edges := elementEdges(elements[next])
			if !isSubset(done, edges) {
				continue
			}
			el := elements[next]
			id := elementID(el)
			wg.Add(1)
			go func(id ID, element T, in []V) {
				defer wg.Done()
				res, err := elementTask(el, ctx, in)
				c <- result[ID, V]{
					ID:    id,
					Value: res,
					Error: err,
				}
			}(id, el, inputs(next, edges))
			next++
		}
	}
	return nil
}

func isSubset[T cmp.Ordered](set, subset []T) bool {
	for _, v := range subset {
		if !slices.Contains(set, v) {
			return false
		}
	}
	return true
}
