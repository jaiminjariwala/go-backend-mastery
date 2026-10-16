package contextbasics

import (
	"context"
	"net/http"
	"time"
)

// Stop closes the entrance, then gives active requests time to finish.
func Stop(server *http.Server, limit time.Duration) error {
	// A fresh context is needed: the signal context is already canceled.
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		server.Close() // Time budget expired: force remaining connections closed.
		return err
	}
	return nil
}
