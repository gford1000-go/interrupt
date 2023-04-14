[![Go Doc](https://pkg.go.dev/badge/github.com/gford1000-go/interrupt.svg)](https://pkg.go.dev/github.com/gford1000-go/interrupt)
[![Go Report Card](https://goreportcard.com/badge/github.com/gford1000-go/interrupt)](https://goreportcard.com/report/github.com/gford1000-go/interrupt)

interrupt
=========

interrupt provides a channel based approach to gracefully handle interrupts.

## Use

Each time a `context` is created, `NewManager` can be called which returns 
a context specific `Manager`.  

The `Manager` allows channels to be added which will be signalled if the associated context completes or an interrupt occurs.

This allows functions to select against a `chan<- bool` rather than more complex case statements, for graceful shutdowns across multiple goroutines.

```go
func launchWorkers(ctx context.Context, n int) []chan any {
	m := NewManager(ctx)
	workers = make([]chan any, n)
	for i := 0; i < n; i++ {
		ch := make(chan any)
		workers[i] = ch

		go func(work chan any) {
			c := make(chan<- bool, 1)
			defer func() {
				// Tidy on goroutine exit
				m.Remove(c)
				close(c)
			}()

			// c will receive a notification when a context event
			// or interrupt occurs
			m.Add(c)

			for {
				select {
				case <-c:
					return
				case task := <-work:
					// complete task
					// ...
				}
			}
		}(ch)
	}
	return workers
}

func main() {
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	workers := launchWorkers(ctx ,100)
	// start issuing tasks to workers ...
}
```

## How?

This command line is all you need.

```
go get github.com/gford1000-go/interrupt
```
