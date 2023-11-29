package topo

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
)

type TaskFunc[T, V any] func(T, context.Context, []V) (V, error)

func Tasks[ID cmp.Ordered, T, V any](ctx context.Context, elements []T, id IdentifierFunc[T, ID], edges EdgeFunc[T, ID], task TaskFunc[T, V]) ([]V, error) {
	if err := Sort(elements, id, edges); err != nil {
		return nil, err
	}
	nodes := make([]node[V], len(elements))
	indexes := make(map[ID]int, len(nodes))
	for i := range nodes {
		indexes[id(elements[i])] = i
	}
	for i := range nodes {
		nodes[i].all = &nodes
		nodes[i].index = i
		nodes[i].done = make(chan struct{})
		nodes[i].function = closure(elements[i], task)
		inputs := edges(elements[i])
		nodes[i].inputs = make([]int, len(inputs))
		for j, inputID := range inputs {
			nodes[i].inputs[j] = indexes[inputID]
		}
	}
	return run(ctx, nodes)
}

func closure[T, V any](element T, task TaskFunc[T, V]) func(ctx context.Context, in []V) (V, error) {
	return func(ctx context.Context, in []V) (V, error) {
		return task(element, ctx, in)
	}
}

func waitForInputs[T any](ctx context.Context, node node[T], all []node[T]) *sync.WaitGroup {
	wg := new(sync.WaitGroup)
	for _, input := range node.inputs {
		wg.Add(1)
		go func(n int, c <-chan struct{}) {
			defer wg.Done()
			for {
				select {
				case _, more := <-c:
					if !more {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(input, all[input].done)
	}
	return wg
}

func run[T any](ctx context.Context, nodes []node[T]) ([]T, error) {
	for i := range nodes {
		nodes[i].wg = waitForInputs(ctx, nodes[i], nodes)
	}
	wg := sync.WaitGroup{}
	for i := range nodes {
		if err := ctx.Err(); err != nil {
			break
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nodes[i].run(ctx)
		}(i)
	}
	wg.Wait()
	results := make([]T, len(nodes))
	errList := make([]error, 0, len(nodes))
	for i := range nodes {
		results[i] = nodes[i].result
		if err := nodes[i].err; err != nil {
			var (
				fnErr functionError
				upErr upstreamError
			)
			switch {
			case errors.As(err, &upErr):
				err = fmt.Errorf("node[%d] skipped due to upstream error", i)
			case errors.As(err, &fnErr):
				err = fmt.Errorf("node[%d] function returned error: %w", i, fnErr.err)
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				err = fmt.Errorf("node[%d] skipped: %w", i, err)
			}
			errList = append(errList, err)
		}
	}
	if len(errList) > 0 {
		return nil, errors.Join(errList...)
	}
	return results, ctx.Err()
}

type node[T any] struct {
	all      *[]node[T]
	index    int
	inputs   []int
	function func(ctx context.Context, in []T) (T, error)
	result   T
	err      error

	done chan struct{}
	wg   *sync.WaitGroup
}

func (t *node[T]) run(ctx context.Context) {
	defer close(t.done)
	t.wg.Wait()
	if err := ctx.Err(); err != nil {
		t.err = err
		return
	}
	in := make([]T, len(t.inputs))
	for i, input := range t.inputs {
		in[i] = (*t.all)[input].result
		if err := (*t.all)[input].err; err != nil {
			t.err = upstreamError{index: i, err: err}
			return
		}
	}
	res, err := t.function(ctx, in)
	if err != nil {
		t.err = functionError{err: err}
	}
	t.result = res
}

type upstreamError struct {
	index int
	err   error
}

func (e upstreamError) Error() string {
	return fmt.Sprintf("upstream error input[%d]: %v", e.index, e.err)
}

func (e upstreamError) Unwrap() error {
	return e.err
}

type functionError struct {
	err error
}

func (e functionError) Error() string {
	return e.err.Error()
}

func (e functionError) Unwrap() error {
	return e.err
}
