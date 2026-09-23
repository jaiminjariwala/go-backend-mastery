package main
import "fmt"

/*

	The backend scenario: an email dispatch queue. A producer drops a batch of emails into a queue, worker goroutine drain it at their own pace, and the producer closes the queue to say "that's everything."

	unbuffered = direct handoff: imagine you are handing a letter to a friend. You must stand there holding the letter, and your friend must be standing right there with their hand out ready to take it. If they aren't ready, you are stuck standing there waiting. That's why unbuffered channels block immediately if no one is listening.

	buffered = queue: it holds N values before anyone needs to receive. Imagine you have a mailbox with a capacity of 3 letters. You can walk up and drop 3 letters inside even if your friend is not home yet. You don't have to wait for anyone. The mailbox holds the letters for you. Once the mailbox is full (capacity is 3), then you have to wait until someone takes a letter out.

*/

// queueEmails drops a batch into a buffered queue and returns it.
// This function takes a slice of strings and returns a channel of strings.
func queueEmails(emails []string) chan string {
	// make(chan string, len(emails)): room for every email up front.
	// sends only block when the buffer is FULL.
	// you create a buffered channel by passing a capacity number as the second argument.
	// a channel that holds strings, with a buffer size/mailbox capacity of 3.
	ch := make(chan string, len(emails))

	for _, e := range emails {
		// send each email address into the channel.
		ch <- e		// no blocking: buffer has room, no receiver needed yet
	}
	return ch
}

func demoBuffered() {
	emails := []string{"a@x.com", "b@x.com", "c@x.com"}
	queue := queueEmails(emails)

	// later, a worker drains the queue at its own pace.
	for i := 0; i < len(emails); i++ {
		fmt.Println("sending to:", <-queue)	// pull out items out of the channel one by one using receive arrow (<-)
	}
}

/*

	Why use Buffered Channels in a Backend?
	-> Think of an email notification system:
	-> A user signs up, and your backend needs to send 5 different welcome emails.
	-> You don't want to make the user wait on the website while 5 emails slowly send over the internet one by one.
	-> Instead, your code dumps the 5 emails into a buffered channel (the queue) instantly.
	-> Background worker goroutines can quietly pull emails out of that queue and send them whenever they have time, while the user's web page loads instantly.

*/