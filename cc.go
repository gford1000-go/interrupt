package interrupt

import (
	"context"
	"sync"
	"time"
)

type submission struct {
	s chan<- bool
	r chan<- bool
}

// contextualChannels manages the addition and removal
// of channels that are affected by a given context
type contextualChannels struct {
	lck      sync.Mutex
	valid    bool
	a        chan *submission
	r        chan *submission
	shutdown chan bool
}

func (c *contextualChannels) close() {
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.valid {
		// Mark instance as invalid and signal shutdown
		c.valid = false
		c.shutdown <- true
	}
}

func (c *contextualChannels) isValid() bool {
	c.lck.Lock()
	defer c.lck.Unlock()

	return c.valid
}

func (c *contextualChannels) init(ctx context.Context) {
	c.a = make(chan *submission, 10)
	c.r = make(chan *submission, 10)
	c.shutdown = make(chan bool, 1)
	c.valid = true

	go func() {
		chs := []chan<- bool{}

		adder := func(sub *submission) {
			found := false
			for _, c := range chs {
				if c == sub.s {
					found = true
					break
				}
			}
			if !found {
				chs = append(chs, sub.s)
			}
			sub.r <- true
		}

		remover := func(sub *submission) {
			for i, ch := range chs {
				if ch == sub.s {
					chs = append(append([]chan<- bool{}, chs[:i]...), chs[i+1:]...)
					break
				}
			}
			sub.r <- true
		}

		defer func() {
			// There is a possible race condition where goroutines
			// are blocked waiting to add/remove chans from chs, at the
			// time of a context completing or interrupt.

			// So ensure that all have a chance to be added or
			// removed prior to sending any notifications

			drain := func(c chan *submission, f func(*submission)) {
				continueSelect := true
				for continueSelect {
					select {
					case sub := <-c:
						f(sub)
					case <-time.After(time.Millisecond):
						continueSelect = false
					}
				}
				close(c) // Prevents further submissions
			}
			drain(c.a, adder)
			drain(c.r, remover)

			// Can now assume all channels are in chs,
			// so can action notification
			for _, c := range chs {
				// Start goroutines in case channel blocks
				// Note the ch are not owned, so are not closed
				go func(ch chan<- bool) {
					ch <- true
				}(c)
			}

			close(c.shutdown)
		}()

		// Listen for additions or removals, plus
		// context completion
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.shutdown:
				return
			case sub := <-c.a:
				adder(sub)
			case sub := <-c.r:
				remover(sub)
			}
		}
	}()
}

func (c *contextualChannels) submit(ch chan<- *submission, subCh chan<- bool) (valid bool) {
	defer func() {
		// In the case where the subCh has been closed, a panic is triggered;
		// then the submission will fail and submit() should return false
		if r := recover(); r != nil {
			valid = false
		}
	}()

	// Make the submission if the object is currently valid
	// Accept that there is an inherent race condition here; if the
	// subCh is closed due to a shutdown / context completion, then
	// this is caught when the submission is attempted to be added
	// to the closed chan
	valid = c.isValid()
	if valid {
		r := make(chan bool, 1)
		ch <- &submission{s: subCh, r: r}
		<-r
		close(r)
	}

	return valid
}

func (c *contextualChannels) add(ch chan<- bool) bool {
	return c.submit(c.a, ch)
}

func (c *contextualChannels) remove(ch chan<- bool) bool {
	return c.submit(c.r, ch)
}
