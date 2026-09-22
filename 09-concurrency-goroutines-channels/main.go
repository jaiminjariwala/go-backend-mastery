package main

func main() {
	demoSequential()
	demoConcurrent()
	demoChannel()

	// uncomment to watch Go catch a deadlock live
	// demoDeadlock()
}