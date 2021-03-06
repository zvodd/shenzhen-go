// The queue command was automatically generated by Shenzhen Go.
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

var _ = runtime.Compiler

func Generate_numbers(output chan<- int) {

	for i := 0; i < 40; i++ {
		output <- i
		<-time.After(time.Millisecond)
	}
	close(output)
}

func Print_survivors(input <-chan int) {

	for range time.Tick(2 * time.Millisecond) {
		in, open := <-input
		if !open {
			break
		}
		fmt.Println(in)
	}
}

func Queue(drop chan<- int, input <-chan int, output chan<- int) {
	const maxItems = 10
	defer func() {
		close(output)
		if drop != nil {
			close(drop)
		}
	}()

	queue := make([]int, 0, maxItems)
	for {
		if len(queue) == 0 {
			if input == nil {
				break
			}
			queue = append(queue, <-input)
		}
		idx := len(queue) - 1
		out := queue[idx]
		select {
		case in, open := <-input:
			if !open {
				input = nil
				break // select
			}
			queue = append(queue, in)
			if len(queue) <= maxItems {
				break // select
			}
			// Drop least-recently read item, but don't block.
			select {
			case drop <- queue[0]:
			default:
			}
			queue = queue[1:]
		case output <- out:
			queue = queue[:idx]
		}
	}
}

func main() {

	channel0 := make(chan int, 0)
	channel1 := make(chan int, 0)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		Generate_numbers(channel0)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Print_survivors(channel1)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Queue(nil, channel0, channel1)
		wg.Done()
	}()

	// Wait for the various goroutines to finish.
	wg.Wait()
}
