/*
	backend scenario: you're adding a request counter to your API - how many hits did /users and /orders get? In a real server every request runs in its own goroutine, so 100 concurrent requests = 100 goroutines sharing one map[string][int]. That's exactly where things explode without a lock.

	-> In Go, standard maps "map[string]int" are not thread-safe.
	-> Imagine a shared whiteboard in an office. If 2 people walk up to the whiteboard at the exact same millisecond and try to erase and write the number of hits for /users, the data gets corrupted.

	go func() {
		for i := 0; i < 1000; i++ {
			hits["/users"]++ // Goroutine A is writing to the map
		}
	}()
	go func() {
		for i := 0; i < 1000; i++ {
			hits["/orders"]++ // Goroutine B is writing to the map at the same time
		}
	}()

	-> When two background tasks try to write to a raw Go map simultaneously, Go's runtime panics and kills the program with this exact error: "fatal error: concurrent map writes"

	-> To fix this, we need a way to ensure only one goroutine can touch the map at a time. That is what a Mutex does.

	-> The Solution — The Bathroom Key (sync.Mutex)
	-> A Mutex (short for mutual exclusion) is like the single key to a bathroom.
*/

package main

import (
	"fmt"
	"sync"
)


func main() {
	var mu sync.Mutex	// the lock.
	count := 0			// the shared variable everyone wants to touch
	
	done := make(chan bool)		// the "I'm finished" signal

	// 100 goroutines, each adds 1 to the SAME counter
	for i := 0; i < 100; i++ {
		go func() {
			// mu protects count so 2 goroutines don't mess it up at the same time.
			mu.Lock()	// I'm touching count now, everyone else wait
			count++		// safely add 1 to the shared count: only one goroutine here at a time
			mu.Unlock()	// done, next: give the key back

			// done is an unbuffered channel used by the workers to report back to the main function.
			done <- true	// drop a true into the channel indicating I'm finished
		}()
	}

	// The main function loops 100 times, pulling items out of the done channel (<-done).
	for i := 0; i<100; i++ {
		<- done	// wait for a signal 100 times 
	}

	// Once all 100 signals are collected, the main function proceeds safely to the end and prints count: 100
	fmt.Println("count:", count)
}


// That's literally all a mutex is: two lines around the shared variable. Everything between Lock() and Unlock() runs one-at-a-time.