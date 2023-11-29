package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

func main() {
	nodes := []node[int]{
		{inputs: []int{}, function: identity(5)},
		{inputs: []int{}, function: fail(fmt.Errorf("banana"))},
		{inputs: []int{0, 1}, function: sum},
		{inputs: []int{2}, function: sum},
	}
	results, err := run(nodes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(results)
}

func sum(in []int) (int, error) {
	var s int
	for _, i := range in {
		s += i
	}
	return s, nil
}

func product(in []int) (int, error) {
	p := 1
	for _, i := range in {
		p *= i
	}
	return p, nil
}

func identity(n int) func([]int) (int, error) {
	return func([]int) (int, error) {
		return n, nil
	}
}

func fail(err error) func([]int) (int, error) {
	return func([]int) (int, error) {
		return 0, err
	}
}

func waitingTask[T any](node node[T], all []node[T]) *sync.WaitGroup {
	wg := new(sync.WaitGroup)
	for _, input := range node.inputs {
		wg.Add(1)
		go func(n int, c <-chan struct{}) {
			for range c {
			}
			wg.Done()
		}(input, all[input].done)
	}
	return wg
}

func run[T any](nodes []node[T]) ([]T, error) {
	for i := range nodes {
		nodes[i].index = i
		nodes[i].done = make(chan struct{})
	}
	for i := range nodes {
		nodes[i].all = &nodes
		nodes[i].wg = waitingTask(nodes[i], nodes)
	}
	wg := sync.WaitGroup{}
	for i := range nodes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nodes[i].run()
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
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				errList = append(errList, fmt.Errorf("node[%d] skipped context done: %w", i, err))
			case errors.As(err, &upErr):
				errList = append(errList, fmt.Errorf("node[%d] skipped due to upstream error", i))
			case errors.As(err, &fnErr):
				errList = append(errList, fmt.Errorf("node[%d] function returned error: %w", i, fnErr.err))
			default:
				errList = append(errList, fmt.Errorf("node[%d] unknown error: %w", i, err))
			}
		}
	}
	if len(errList) > 0 {
		return nil, errors.Join(errList...)
	}
	return results, nil
}

type node[T any] struct {
	all      *[]node[T]
	index    int
	inputs   []int
	function func(in []T) (T, error)
	result   T
	err      error

	done chan struct{}
	wg   *sync.WaitGroup
}

func (t *node[T]) run() {
	defer close(t.done)
	t.wg.Wait()
	in := make([]T, len(t.inputs))
	for i, input := range t.inputs {
		in[i] = (*t.all)[input].result
		if err := (*t.all)[input].err; err != nil {
			t.err = upstreamError{index: i, err: err}
			return
		}
	}
	res, err := t.function(in)
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
