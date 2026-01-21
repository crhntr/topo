package topo

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type TaskFunc[T, V any] func(T, context.Context, []V) (V, error)

func Tasks[ID comparable, T, V any](ctx context.Context, elements []T, id IdentifierFunc[T, ID], edges EdgeFunc[T, ID], task TaskFunc[T, V]) ([]V, error) {
	elementID := id
	type TaskState uint8

	const (
		Initializing TaskState = iota
		Waiting
		Loading
		Running
		Done
		Errored
		Skipped
	)

	type run struct {
		state TaskState
		sync.WaitGroup
		result V
		err    error
	}

	done := func(node *graphNode[T, ID, run]) {
		for _, c := range node.children {
			c.data.Done()
		}
	}

	execRun := func(node *graphNode[T, ID, run]) {
		if err := ctx.Err(); err != nil {
			node.data.state = Skipped
			done(node)
		}
		node.data.state = Waiting
		node.data.Wait()
		node.data.state = Loading

		var inputs []V
		for _, p := range node.parents {
			if p.data.state == Errored || p.data.state == Skipped {
				node.data.state = Skipped
				done(node)
			}
			inputs = append(inputs, p.data.result)
		}

		node.data.state = Running
		result, err := task(node.element, ctx, inputs)
		if err != nil {
			node.data.state = Errored
		} else {
			node.data.state = Done
		}
		node.data.err = err
		node.data.result = result
		done(node)
	}

	wg := sync.WaitGroup{}

	errC := make(chan error)

	resultsLock := new(sync.Mutex)
	results := make([]V, len(elements))
	_, err := newGraph(elements, elementID, edges, func(n *graphNode[T, ID, run]) {
		n.data.state = Initializing
		for range n.parents {
			n.data.Add(1)
		}
		node := n
		wg.Go(func() {
			defer func() {
				r := recover()
				if r != nil {
					errC <- fmt.Errorf("recovered: %s", r)
				}
			}()
			n := node
			execRun(n)
			resultsLock.Lock()
			results[n.index] = n.data.result
			resultsLock.Unlock()
			if n.data.err != nil {
				errC <- n.data.err
			}
		})
	})

	go func() {
		defer close(errC)
		wg.Wait()
	}()

	for e := range errC {
		err = errors.Join(err, e)
	}

	return nil, errors.Join(err, ctx.Err())
}
