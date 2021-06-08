package rungroup

import (
	"context"
	"sync"
)

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero Group is valid and does not cancel on error.
// The errMap contains errors for all goroutines where key is id of goroutine.
type Group struct {
	cancel func()

	wg sync.WaitGroup

	errOnce sync.Once
	err     error
	errMap  map[string]error
}

// WithContext returns a new Group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go or GoWithFunc
// returns a non-nil error on interruptor routine
// or the first time Wait returns, whichever occurs first.
func WithContext(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{cancel: cancel, errMap: make(map[string]error)}, ctx
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from interrupter routines.
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// Go calls the given function in a new goroutine.
//
// The first call to return a non-nil error cancels the group if its interruptor routine ; its error will be
// returned by Wait.
// Interrupter is a flag signifies if a goroutine can interrupt other goroutines in group.
func (g *Group) Go(f func() error, interrupter bool, id string) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		if err := f(); err != nil {
			g.errMap[id] = err
			if interrupter {
				g.errOnce.Do(func() {
					g.err = err
					if g.cancel != nil {
						g.cancel()
					}
				})
			}
		}
	}()
}

// GetErrById returns the error associated with goroutine id
func (g *Group) GetErrById(id string) error {
	return g.errMap[id]
}

// GoWithFunc is a closure over a normal input func.
// This is done to ensure that all goroutines wait for error(func execution)
// or context cancellation.
func (g *Group) GoWithFunc(f func(ctx context.Context) error,
	ctx context.Context,
	interrupter bool, id string) {

	gFunc := func() error {

		errChan := make(chan error, 1)
		closure := func(errChan chan<- error) {

			defer func() {
				close(errChan)
			}()

			err := f(ctx)
			if err != nil {
				errChan <- err
				return
			}
		}

		closure(errChan)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			return err
		}

	}

	g.Go(gFunc, interrupter, id)
}
