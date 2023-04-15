package interrupt

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

var m *manager

func init() {
	m = &manager{
		m: map[context.Context]*contextualChannels{},
	}

	go func() {
		// Wait for CTRL-C interrupt
		ctrlC := make(chan os.Signal, 1)
		signal.Notify(ctrlC, os.Interrupt)
		<-ctrlC

		// Interrupted, so signal to all channels
		m.close()
	}()
}

// manager will send a signal to all the
// channels it is aware of, when an interrupt is received
type manager struct {
	m map[context.Context]*contextualChannels
	l sync.Mutex
}

func (i *manager) close() {
	// Signals all channels in all contexts
	m.l.Lock()
	defer m.l.Unlock()

	for _, v := range m.m {
		v.close()
	}
	m.m = map[context.Context]*contextualChannels{}
}

func (i *manager) findOrCreate(ctx context.Context) *contextualChannels {
	i.l.Lock()
	defer i.l.Unlock()

	cc, ok := i.m[ctx]
	if !ok {
		cc = &contextualChannels{}
		cc.init(ctx)
		i.m[ctx] = cc
	}

	return cc
}

// add will associate the specified channel as a channel to be
// notified for context events and general interrupts
func (i *manager) add(ctx context.Context, c chan<- bool) bool {
	cc := i.findOrCreate(ctx)
	return cc.add(c)
}

// remove disassociates the specified channel as a channel to be
// notified for context events and general interrupts
func (i *manager) remove(ctx context.Context, c chan<- bool) bool {
	cc := i.findOrCreate(ctx)
	return cc.remove(c)
}

// Manager provides a context specific approach to obtaining
// notifications via channels when either the context completes,
// or when an interrupt is detected.
//
// An arbitrary number of channels can be added to the manager,
// ephemerally or permanently, and all will receive notifications
// to exit without needing to be aware of their origin
type Manager struct {
	ctx context.Context
}

// Add will associate the specified channel as a channel to be
// notified for context events and general interrupts
func (i *Manager) Add(c chan<- bool) bool {
	return m.add(i.ctx, c)
}

// Remove disassociates the specified channel as a channel to be
// notified for context events and general interrupts
func (i *Manager) Remove(c chan<- bool) bool {
	return m.remove(i.ctx, c)
}

// NewManager can be called as often as needed as new contexts
// are created.  Context based changes are limited to the channels in
// each context, but an interrupt will be sent to all channels of
// all contexts
func NewManager(ctx context.Context) *Manager {
	return &Manager{ctx: ctx}
}

// GetContextInterruptNotfier provides a single call to create
// a channel that will receive an event when either the context
// completes or an interrupt occurs.
// The channel has been created successfully only if the function returns true.
func GetContextInterruptNotfier(ctx context.Context) (<-chan bool, bool) {
	c := make(chan bool, 1)

	r := make(chan bool, 1)
	defer close(r)

	go func() {
		shutdown := make(chan bool, 1)
		defer close(shutdown)

		m := NewManager(ctx)

		addedOk := m.Add(shutdown)
		r <- addedOk

		if addedOk {
			// Only if Add returns true has the shutdown channel
			// been added successfully, so can now wait for notifications
			defer func() {
				m.Remove(shutdown)
			}()

			// Only interrupts or context events are captured and
			// pushed to shutdown, which may therefore exit sooner
			<-shutdown

			// Notify to external party
			c <- true
		}

		close(c)
	}()

	return c, <-r
}
