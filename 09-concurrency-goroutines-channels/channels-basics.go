package main
import (
	"fmt"
	"time"
)


// checkServiceV2 is checkService, but it SENDS its result back instead of printing.
// "ch chan string" reads as: ch is a channel that carries strings.
func checkServiceV2(name string, ch chan string) {
	time.Sleep(500 * time.Millisecond)
	ch <- name + " is up"	// SEND: the arrow points INTO the channel
}



func demoChannel() {
	ch := make(chan string)	// make a channel that carries strings

	// launch 3 background goroutines and hand each of them the exact same channel "ch"
	// the work: All 3 goroutines sleep for 500ms in the background.
	go checkServiceV2("auth", ch)
	go checkServiceV2("payments", ch)
	go checkServiceV2("search", ch)

	// RECEIVE 3 results "<-ch" reads as: take a value OUT of the channel.
	// Each receive BLOCKS until a value arrives: main waits here patiently.
	// Channels are FIFO: first sent in = first received out.

	// Channels are blocking. When the main program tries to read from an empty channel(<-ch), it stops dead in its tracks and waits patiently until someone sends something.
	// 	As each of the 3 background goroutines finishes its 500ms sleep, it executes ch <- name + " is up".
	// The moment a message drops into the channel, the main program unblocks, grabs the message, prints it (fmt.Println(result)), and loops to get the next one.
	for i:=0; i<3; i++ {
		result := <-ch
		fmt.Println(result)
	}
	// No sleep hack needed: the receives ARE the waiting.
	// This is the fan-out pattern: one request fans out to N services,
	// then gathers every result. Used constantly in backends.
}




/*
	What is a deadlock?
	-> a deadlock happens when your program gets stuck in a state where it is waiting for an event that can never possibly happen, causing Go to panic and crash the program.
*/
func demoDeadlock() {
	ch := make(chan string)	// make a channel that carries strings

	// SEND blocks until someone receives. Nobody ever receives.
	// Go detects the standstill and panics: "all goroutines are asleep - deadlock!"
	ch <- "hello"	// stuck HERE forever
	fmt.Println(<-ch)
}



/*
	In previous example, your background goroutines (checkService) could do work, but they couldn't easily send their results back to the main program. They just printed to the console and vanished.

	-> A channel is a typed communication pipe that connects goroutines.
	-> One goroutine can drop a piece of data into one end, and another goroutine can pick it up from the other end.
	-> "ch chan string" means: "A channel (ch) that only passes strings (string)"

	-> The arrow (<-) is the visual indicator of data flow. It always points in the direction the data is moving.
	-> Sending (ch <- value): The arrow points into the channel. You are pushing data into the pipe.
	-> Receiving (<- ch): The arrow points away from the channel (or the channel is on the right). You are pulling data out of the pope.

*/



// By default, Go channels are unbuffered. This means they cannot store data. When you send data into an unbuffered (ch<-"hello"), the sender pauses and blocks until another goroutine is actively standing by to receive it.