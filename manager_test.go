package interrupt

import (
	"context"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	// Managers are associated to a single context
	m1 := NewManager(ctx1)
	m2 := NewManager(ctx2)

	// This captures the channels that have started
	sig := []string{}
	var lck sync.Mutex

	f := func(n string) (chan<- bool, <-chan bool) {
		c := make(chan bool, 1)
		h := make(chan bool, 1)
		go func() {
			<-c
			lck.Lock()
			defer lck.Unlock()
			sig = append(sig, n)
			h <- true
		}()
		return c, h
	}

	// Start some channels
	c1, h1 := f("c1")
	c2, h2 := f("c2")
	c3, h3 := f("c3")

	// Store expected results
	expA := []string{}
	expB := []string{}

	// Attempt to add channels - this is done in a separate
	// goroutine so there is a race condition here between
	// addition of the channels and progress on this main goroutine
	if m1.Add(c1) {
		expA = append(expA, "c1")
	}
	if m2.Add(c2) {
		expB = append(expB, "c2")
	}
	if m1.Add(c3) {
		expA = append(expA, "c3")
	}

	sigtest := func(expected []string) {
		lck.Lock()
		defer lck.Unlock()
		for _, e := range expected {
			found := false
			for _, f := range sig {
				if e == f {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("unexpected result: wanted %s, not found", e)
			}
		}
	}

	// Should trigger expA containing a max of "c1" and "c3",
	// but this will vary by race condition across the goroutines
	cancel1()

	// Ensure that the goroutines have a chance to run
	<-h1
	<-h3

	// We should have a match between what has been added to
	// m1 and the channels that are were listening
	sigtest(expA)

	// Do same for the second context
	cancel2()
	<-h2
	sigtest(expB)
}
