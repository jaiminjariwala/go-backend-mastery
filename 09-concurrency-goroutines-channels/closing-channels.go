package main
import "fmt"

// close(ch) is how a sender says "nothing more is coming, ever."
// Golden rule: only the SENDER closes. Never the receiver.
// Sending on a closed channel panics. Closing is optional 


// dispatchEmails sends a batch, then closes: no more emails after this.
// "ch chan<- string" reads as: ch is a SEND-ONLY channel of strings
// The compiler forbids this function from receiving on ch.
// means it tells compiler: "Inside this function, I am only allowed to push data into 'ch'. If I accidentally try to read from it, crash the code."
func dispatchEmails(ch chan<- string, emails []string) {
	for _, e := range emails {
		ch <- e
	}

	// why close? this is the sender's way of saying: "I am completely finished sending emails. No more data will ever come through this pipe."
	close(ch)
}


// worker drains emails until the channel is closed.
// "ch <-chan string" reads as: ch is a RECEIVE-ONLY channel of strings.
// The compiler forbids this function from sending or closing ch.
func worker(ch <-chan string) {
	// range over a channel: keeps receiving until the channel is CLOSED,
	// then the loop ends by itself. No counting needed.
	for e := range ch {
		fmt.Println("worker sending to:", e)
	}
	fmt.Println("worker: channel closed, going home")
}


func demoCloseRange() {
	// create an empty channel (pipe) that can hold up to 10 strings
	ch := make(chan string, 10)

	emails := []string{"a@x.com", "b@x.com", "c@x.com"}

	// because of "go" keyword, go spins up a background helper task. this helper immediately starts running dispatchEmails in the background
	go dispatchEmails(ch, emails)

	// the main program immediately moves to the next line and calls worker function. the main thread is now stuck inside the worker function, waiting.
	worker(ch)	// blocks inside range until dispatchEmails closes ch
}


func demoOkIdion() {
	ch := make(chan string, 1)
	ch <- "one last email"
	close(ch)

	// "v, ok := <-ch" reads as: receive a value; ok is false
	// when the channel is closed AND empty.
	v, ok := <-ch
	fmt.Println(v, ok)

	v, ok = <-ch
	fmt.Println(v, ok)
}