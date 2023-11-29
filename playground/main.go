package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	nodes := []node{
		{InputIndexes: []int{}, Function: func() {
			fmt.Println("init", 0)
			<-time.After(time.Second)
			fmt.Println("close", 0)
		}},
		{InputIndexes: []int{0}, Function: func() {
			fmt.Println("init", 1)
			<-time.After(time.Second)
			fmt.Println("close", 1)
		}},
		{InputIndexes: []int{0}, Function: func() {
			fmt.Println("init", 2)
			<-time.After(time.Second)
			fmt.Println("close", 2)
		}},
		{InputIndexes: []int{0}, Function: func() {
			fmt.Println("init", 3)
			<-time.After(time.Second)
			fmt.Println("close", 3)
		}},
	}
	run(nodes)
}

func waitingTask(node node, all []node) *sync.WaitGroup {
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

func run(nodes []node) {
	for i := range nodes {
		nodes[i].index = i
		nodes[i].done = make(chan struct{})
	}
	for i := range nodes {
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
}

type node struct {
	index        int
	InputIndexes []int
	Function     func()

	done   chan struct{}
	inputs *sync.WaitGroup
}

func (t *node) run() {
	defer close(t.done)
	t.inputs.Wait()
	t.Function()
}
