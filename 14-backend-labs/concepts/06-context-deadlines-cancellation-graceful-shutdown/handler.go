package contextbasics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Wait represents slow work that cooperates with cancellation.
func Wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err() // Cancellation or deadline: stop early.
	case <-timer.C:
		return nil // Work finished.
	}
}

// Task returns a handler; durations are fixed by the server, not user input.
func Task(workTime, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		log.Println("lesson 6: task started")
		err := Wait(ctx, workTime)
		if err == context.Canceled {
			log.Println("lesson 6: client canceled; task stopped early")
			return // The browser stopped waiting; no useful response to send.
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err == context.DeadlineExceeded {
			log.Println("lesson 6: deadline reached; task stopped early")
			w.WriteHeader(408) // This demo's chosen timeout response.
			fmt.Fprintln(w, "Deadline reached: the server stopped the task.")
			return
		}
		log.Println("lesson 6: task completed")
		fmt.Fprintln(w, "Task completed successfully.")
	}
}
