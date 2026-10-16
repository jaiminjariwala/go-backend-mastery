package contextbasics

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestShutdownWaitsForActiveRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.Write([]byte("finished"))
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	go server.Serve(listener)
	clientDone := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			defer response.Body.Close()
			var body []byte
			body, err = io.ReadAll(response.Body)
			if string(body) != "finished" {
				err = io.ErrUnexpectedEOF
			}
		}
		clientDone <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request never started")
	}
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- Stop(server, 3*time.Second) }()
	select {
	case err := <-shutdownDone:
		t.Fatal("shutdown must wait for active request", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-clientDone; err != nil {
		t.Fatal(err)
	}
	if err := <-shutdownDone; err != nil {
		t.Fatal(err)
	}
}
