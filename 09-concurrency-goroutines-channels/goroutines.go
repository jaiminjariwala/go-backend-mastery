// The backend scenario: your API needs to check 3 downstream services (auth, payments, search) on every request. Call them one by one and the latencies add up. Call them concurrently and you only wait for the slowest one.

// What is a downstream service ?
/* 
	A downstream service is any external system, database, microservice, or API that your application depends on to do its job.

	Think of data flowing like water in a river. Your main application sits upstream, receiving the initial request from the user. To fulfil that request, it has to reach out to other systems further down the line - its downstream dependencies.
*/

package main
import (
	"fmt"
	"time"
)

// checkService fakes a call to a downstream service.
/*
	What does "faking a call" mean ?
	-> A real call would mean your Go program has to open a network connection, send an HTTP request accross the internet or local network to another server (e.g., [https://api.paymentprovider.com/verify](https://api.paymentprovider.com/verify)), wait for that server to process it, and receive a response.

	-> So faking a call means the code simulates the network trip without actually talking to a real external server.
	-> Instead of making a real network request, it uses time.Sleep(500 * time.Milliseond).
	-> This mimics network latency - the physical time it takes for data to travel back and forth over a network and for the remote server to process it.
*/

// this function takes a service name and returns nothing.
// the Sleep fakes network latency: each call takes 500ms.
func checkService(name string) {
	time.Sleep(500 * time.Millisecond)
	fmt.Println(name, "is up")
}

func demoSequential() {
	start := time.Now()

	checkService("auth")		// runs, finishes, THEN the next line runs
	checkService("payments")	// each call waits for the previous one
	checkService("search")

	// so how much time it waits?
	// 3 x 500ms = ~1.5s. This is how slow a backend gets when a handler calls services one by one (user, then orders, then recommendations).
	fmt.Println("sequential took", time.Since(start))
}

func demoConcurrent() {
	start := time.Now()

	// "go" reads as: run this in the background, don't wait for it.
	// Each go statement spawns a goroutine: a lightweight task run by Go's runtime.
	// Much cheaper than an OS thread: thousands of goroutines is normal.
	go checkService("auth")
	go checkService("payments")
	go checkService("search")

	// all three run at the same time now. But main doesn't wait for them,
	time.Sleep(700 * time.Millisecond)

	// ~0.5s instead of ~1.5s. That gap is the entire point of concurrency.
	fmt.Println("concurrent took:", time.Since(start))
}

/*
	What is an OS thread vs. a Goroutine?
	-> OS Thread: When your computer runs a program, the operating system creates a "thread". Creating an OS thread takes a noticeable amount of memory and time.

	-> Goroutine: A goroutine is not a thread. It is a lightweight task managed by Go. It uses almost zero memory(only a few kilobytes), meaning your computer can run thousands of them at the same time without breaking a sweat. Go's internal system takes these goroutines and maps them onto actual OS threads behind the scenes.


	What does "go" keyword do ?
	-> Normally, Go executes your code line by line, top to bottom. When it hits a function, it pauses, waits for that function to completely finish, and only then moves to the next line.

	-> When you put the "go" keyword in front of a function call, you are telling Go:
	"Start this function in the background, but do not wait for it to finish. Immediately jump to the next line of code."



	Why is time.Sleep(700 * time.Millisecond) at the bottom?
	-> This is a quirk of how Go programs work.
	-> When your main function reaches the end of demoConcurrent(), the program completely exits. If the main program exits, all background goroutines are instantly killed, even if they haven't finished printing "is up" yet.
	-> Because the background tasks take 500ms to run, the main program needs to stay alive long enough for them to finish. The time.Sleep(700 * time.Millisecond) is just a temporary pause to keep the main program alive for 0.7 seconds so the background goroutines have time to finish printing their messages before the program shuts down.



	Problem: "go" is fire-and-forget. How do results come back, and how does main know when everything's done? That's what channels are for! File 2: chanels-basics.go.
*/