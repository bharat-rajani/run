package rungroup

import (
	"context"
	"errors"
	"sync"

	"github.com/bharat-rajani/rungroup/pkg/concurrent"
)

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
// A zero Group is valid and does not cancel on error.
// The errMap contains errors for all goroutines where key is id of goroutine.
type Group[K, V any] struct {
	cancel func()

	wg sync.WaitGroup

	errOnce   sync.Once
	err       error
	resultMap concurrent.ConcurrentMap
}

// WithContext returns a new Group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go or GoWithFunc
// returns a non-nil error on interrupter routine
// or the first time Wait returns, whichever occurs first.

// WithContext creates a new concurrent task Group with a specified context.
// It accepts the following parameter:
//
//   - ctx: The parent context in which the Group and its tasks will operate.
//
// The function returns two values:
//
//   - A pointer to a new Group[K, V] instance initialized with the provided context.
//   - A new context.Context derived from the input context 'ctx' with an associated cancellation function.
//     This context is canceled when the first interrupter goroutine in the group returns a non-nil error.
//
// This function is useful for setting up a concurrent task Group with a specific context,
// allowing you to manage the lifecycle and behavior of the tasks within that context.
func WithContext[K, V any](ctx context.Context) (*Group[K, V], context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group[K, V]{cancel: cancel}, ctx
}

// WithContextResultMap creates a new concurrent task Group with a specified context and result map.
// It accepts the following parameters:
//
//   - ctx: The parent context in which the Group and its tasks will operate.
//   - resultMap: A ConcurrentMap where the results of concurrent tasks will be stored.
//
// The function returns two values:
//
//   - A pointer to a new Group[K, V] instance initialized with the provided context and resultMap.
//   - A new context.Context derived from the input context 'ctx' with an associated cancellation function.
//
// This function is useful for setting up a concurrent task Group with a specific context and resultMap
// for tracking and managing the results of concurrent tasks.
func WithContextResultMap[K, V any](ctx context.Context, resultMap concurrent.ConcurrentMap) (*Group[K, V], context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Group[K, V]{cancel: cancel, resultMap: resultMap}, ctx
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from interrupter routines.
func (g *Group[K, V]) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

// Go executes a provided function 'f' concurrently within the Group. It accepts the following parameters:
//
//   - f: The function to be executed concurrently, which should return a result value of type 'V' and an error.
//   - interrupter: A boolean flag indicating whether the function execution can be interrupted upon error.
//   - id: A unique identifier ('id') associated with this concurrent task.
//
// The method creates a new goroutine to execute 'f', and it manages the result and error values.
// If 'f' returns an error, it stores the error in the Group's resultMap if it exists.
// If 'interrupter' is set to 'true', it also records the error at the Group level for potential cancellation.
// If 'f' completes successfully, it stores the result in the resultMap if it exists.
//
// Note: To ensure proper tracking of concurrent tasks and their results, always use 'Go' to execute functions
// within the Group, and associate each task with a unique identifier ('id').
func (g *Group[K, V]) Go(f func() (V, error), interrupter bool, id K) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()

		if v, err := f(); err != nil {
			if g.resultMap != nil {
				var zeroVal V
				g.storeResult(id, zeroVal, err)
			}
			if interrupter {
				g.errOnce.Do(func() {
					g.err = err
					if g.cancel != nil {
						g.cancel()
					}
				})
			}
		} else {
			if g.resultMap != nil {
				g.storeResult(id, v, nil)
			}
		}
	}()
}

// storeResult associates a result ('v') and an error ('err') with a unique identifier ('id')
// in the Group's resultMap. It uses a closure to encapsulate the result and error values
// for later retrieval. This allows for convenient storage and retrieval of concurrent task results.
//
// Parameters:
//   - id: A unique identifier associated with the concurrent task result.
//   - v: The result value of type 'V' to be stored.
//   - err: An error value, if any, associated with the result.
func (g *Group[K, V]) storeResult(id K, v V, err error) {
	g.resultMap.Store(id, func() (V, error) { return v, err })
}

// GetResultByID retrieves the result associated with a specific identifier ('id')
// from the Group's resultMap. It returns the following three values:
//
//   - V: The result value associated with the identifier.
//   - error: An error value, if any, encountered during the retrieval process.
//   - bool: A boolean indicating whether the result was found ('true' if found, 'false' otherwise).
//
// If the Group's resultMap is nil, it returns a zero value of type V, an 'ErrGroupNilMap' error,
// and 'false' to indicate that the result was not found due to the nil map.
//
// If the result is not found in the resultMap, it returns a zero value of type V, a 'nil' error,
// and 'false' to indicate that the result was not found.
//
// If the result is found and successfully retrieved, it returns the result value ('V'),
// 'nil' error, and 'true' to indicate that the result was found.
//
// Note: The 'id' parameter should be a unique identifier associated with a previously executed
// concurrent task in the Group. Ensure that you use 'GoWithFunc' to associate unique identifiers
// with concurrent functions in the Group.
func (g *Group[K, V]) GetResultByID(id K) (V, error, bool) {
	var zeroVal V
	if g.resultMap == nil {
		return zeroVal, ErrGroupNilMap, false
	}

	val, ok := g.resultMap.Load(id)
	if !ok {
		return zeroVal, nil, false
	}
	v, err := val.(func() (V, error))()
	return v, err, ok
}

// GoWithFunc runs a given function concurrently and associates it with a unique identifier.
// It accepts the following parameters:
//
//   - f: The function to be executed concurrently. It should take a context.Context
//     and return a value of type V and an error.
//   - ctx: The context in which the function will run, allowing for cancellation and timeout handling.
//   - interrupter: A boolean flag indicating whether the function can be interrupted if the sub-context is canceled.
//   - id: A unique identifier (of type K) associated with this concurrent function.
//
// The method creates a goroutine to execute the function 'f' and manages error handling
// and timeouts using channels. If the context is canceled before the function completes,
// it returns a zero value of type V and the context error. If 'f' encounters an error,
// it returns the zero value of type V and that error. If 'f' completes successfully,
// it returns the result value and nil for the error.
//
// Example usage:
//
//	myGroup := NewGroup[K, V]()
//	myGroup.GoWithFunc(myFunction, context.Background(), false, myIdentifier)
//
// Note: It's essential to use GoWithFunc to execute functions concurrently within a Group.
// The unique identifier 'id' is useful for managing and tracking concurrent tasks.
func (g *Group[K, V]) GoWithFunc(f func(ctx context.Context) (V, error),
	ctx context.Context,
	interrupter bool, id K) {

	gFunc := func() (V, error) {

		errChan := make(chan error, 1)
		resultChan := make(chan V, 1)
		closure := func(erChan chan<- error) {

			defer func() {
				close(errChan)
				close(resultChan)
			}()

			v, err := f(ctx)
			if err != nil {
				erChan <- err
				return
			} else {
				resultChan <- v
			}
		}

		go closure(errChan)

		var zeroVal V
		select {
		case <-ctx.Done():
			return zeroVal, ctx.Err()
		case err := <-errChan:
			return zeroVal, err
		case v := <-resultChan:
			return v, nil
		}

	}

	g.Go(gFunc, interrupter, id)
}

// ErrGroupNilMap is thrown while calling group.GetErrByID() when Group.errMap is nil
var ErrGroupNilMap = errors.New("uninitialized error map in rungroup, use: WithContextErrMap")
