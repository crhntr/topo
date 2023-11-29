package main

import (
	"fmt"
	"sync"
)

func main() {
	nodes := []node[int]{
		{InputIndexes: []int{}, Function: identity(2)},
		{InputIndexes: []int{}, Function: identity(3)},
		{InputIndexes: []int{0, 1}, Function: product},
		{InputIndexes: []int{4, 0}, Function: product},
		{InputIndexes: []int{0, 1, 2}, Function: sum},
	}
	fmt.Println(run(nodes))
}

func sum(in []int) int {
	var s int
	for _, i := range in {
		s += i
	}
	return s
}

func product(in []int) int {
	p := 1
	for _, i := range in {
		p *= i
	}
	return p
}

func identity(n int) func([]int) int {
	return func([]int) int {
		return n
	}
}

func waitingTask[T any](node node[T], all []node[T]) *sync.WaitGroup {
	wg := new(sync.WaitGroup)
	for _, input := range node.InputIndexes {
		wg.Add(1)
		go func(n int, c <-chan struct{}) {
			for range c {
			}
			wg.Done()
		}(input, all[input].done)
	}
	return wg
}

func run[T any](nodes []node[T]) []T {
	for i := range nodes {
		nodes[i].index = i
		nodes[i].done = make(chan struct{})
	}
	for i := range nodes {
		nodes[i].all = &nodes
		nodes[i].inputs = waitingTask(nodes[i], nodes)
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
	for i := range nodes {
		results[i] = nodes[i].Result
	}
	return results
}

type node[T any] struct {
	all          *[]node[T]
	index        int
	InputIndexes []int
	Function     func(in []T) T
	Result       T

	done   chan struct{}
	inputs *sync.WaitGroup
}

func (t *node[T]) run() {
	defer close(t.done)
	t.inputs.Wait()
	in := make([]T, len(t.InputIndexes))
	for i, input := range t.InputIndexes {
		in[i] = (*t.all)[input].Result
	}
	t.Result = t.Function(in)
}
