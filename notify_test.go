package interrupt

import (
	"context"
	"testing"
	"time"
)

func TestNotify(t *testing.T) {

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Create the chan that will be notified
	c, ok := GetContextInterruptNotfier(ctx)

	if !ok {
		t.Fatal("Failed to create channel")
	}

	pause := make(chan bool, 1)
	defer close(pause)

	received := make(chan bool, 1)
	defer close(received)

	go func() {
		pause <- true // Signal goroutine is running

		select {
		case <-time.After(100 * time.Millisecond):
			received <- false
		case <-c:
			received <- true
		}
	}()

	// Wait for goroutine to start
	<-pause

	// Cancel the context, should trigger notification
	cancel()

	// See what we got...
	if resp := <-received; !resp {
		t.Fatal("timeout rather than receiving notification")
	}
}
