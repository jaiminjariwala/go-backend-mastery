package contextbasics

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	tests := []struct {
		name          string
		work, timeout time.Duration
		status        int
		message       string
	}{
		{"finishes", 0, time.Second, 200, "Task completed"},
		{"times out", time.Hour, 0, 408, "Deadline reached"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			Task(tt.work, tt.timeout)(response, httptest.NewRequest("GET", "/", nil))
			if response.Code != tt.status || !strings.Contains(response.Body.String(), tt.message) {
				t.Error("unexpected response", response.Code, response.Body.String())
			}
		})
	}
}

func TestCancellationStopsWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() { finished <- Wait(ctx, time.Hour) }()
	cancel()
	select {
	case err := <-finished:
		if err != context.Canceled {
			t.Error("expected cancellation, got", err)
		}
	case <-time.After(time.Second):
		t.Fatal("work ignored cancellation")
	}
}

func TestCanceledHandlerDoesNotWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	Task(time.Hour, time.Second)(response, request)
	if response.Body.Len() != 0 {
		t.Error("canceled request should not receive a body")
	}
}
