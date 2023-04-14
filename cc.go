package interrupt

import (
	"context"
	"sync"
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
		c.valid = false
		c.shutdown <- true

		close(c.a)
		close(c.r)
		close(c.shutdown)
	}
}

func (c *contextualChannels) init(ctx context.Context) {
	c.a = make(chan *submission, 1)
	c.r = make(chan *submission, 1)
	c.shutdown = make(chan bool, 1)
	c.valid = true

	go func() {
		chs := []chan<- bool{}
		defer func() {
			for _, c := range chs {
				// Start goroutines in case channel blocks
				go func(ch chan<- bool) {
					ch <- true
				}(c)
			}
			c.close()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.shutdown:
				return
			case sub := <-c.a:
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
			case sub := <-c.r:
				for i, ch := range chs {
					if ch == sub.s {
						chs = append(append([]chan<- bool{}, chs[:i]...), chs[i+1:]...)
						break
					}
				}
				sub.r <- true
			}
		}
	}()
}

func (c *contextualChannels) submit(ch chan<- *submission, subCh chan<- bool) bool {
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.valid {
		r := make(chan bool, 1)
		ch <- &submission{s: subCh, r: r}
		<-r
		close(r)
	}

	return c.valid
}

func (c *contextualChannels) add(ch chan<- bool) bool {
	return c.submit(c.a, ch)
}

func (c *contextualChannels) remove(ch chan<- bool) bool {
	return c.submit(c.r, ch)
}
