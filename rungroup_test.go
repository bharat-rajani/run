package rungroup_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/bharat-rajani/rungroup/pkg/concurrent"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bharat-rajani/rungroup"
)

type testArg struct {
	f           func() (any, error)
	interrupter bool
	id          string
}

type testWant struct {
	id  string
	err error
	ok  bool
}

func prepareRandomTestArgs(n int, pIntr int, pErr int, err error) []testArg {

	if pIntr < 0 || pIntr > 100 {
		pIntr = 0
	}

	if pErr < 0 || pErr > 100 {
		pErr = 0
	}

	testArgs := make([]testArg, n)

	getSleepDuration := func(rand int) time.Duration {
		// 20 percent goroutine will sleep for 300 msec
		// 30 percent goroutine will sleep for 200 msec
		// 50 percent goroutine will sleep for 100 msec

		if rand <= 20 {
			return time.Duration(300) * time.Millisecond
		} else if rand <= 50 {
			return time.Duration(200) * time.Millisecond
		} else {
			return time.Duration(100) * time.Millisecond
		}
	}

	for i := 0; i < n; i++ {
		rand.Seed(time.Now().UnixNano())
		testArgs[i].id = fmt.Sprintf("%d", i)
		ir := rand.Intn(n-1) + 1
		fir := float64(ir) / float64(n)
		// if its interrupter goroutine it must return a specified error (err in args)
		if fir <= (float64(pIntr) / float64(100)) {
			testArgs[i].interrupter = true
			testArgs[i].f = func() (any, error) {
				time.Sleep(getSleepDuration(ir))
				return nil, err
			}
		} else {
			// it may or may not give error
			// rest of the percentage of pErr will go here
			testArgs[i].interrupter = false
			er := rand.Intn(101-1) + 1
			// percent specifies percentage of goroutine returning an error
			if er <= pErr {
				testArgs[i].f = func() (any, error) {
					time.Sleep(getSleepDuration(er))
					return nil, err
				}
			} else {
				testArgs[i].f = func() (any, error) {
					time.Sleep(getSleepDuration(er))
					return nil, nil
				}
			}
		}
	}

	return testArgs
}

func TestWithContext(t *testing.T) {
	errBomb := errors.New("group_test: fBombed")

	cases := []struct {
		name        string
		goRoutineId string
		interrupter bool
		errs        []error
		want        error
	}{
		{name: "NoErrorsWhenAnyIntrpt", interrupter: true, want: nil},
		{name: "NoErrorsWhenNoIntrpt", interrupter: false, want: nil},
		{name: "NilErrorWhenAnyIntrpt", interrupter: true, errs: []error{nil}, want: nil},
		{name: "NilErrorWhenNoIntrpt", interrupter: false, errs: []error{nil}, want: nil},
		{name: "ErrorBombWhenNoIntrpt", interrupter: false, errs: []error{errBomb}, want: nil},
		{name: "ErrorBombWhenAllIntrpt", errs: []error{errBomb, nil}, interrupter: true, want: errBomb},
		{name: "ErrorBombsWhenAllIntrptCheckDataRace", errs: []error{errBomb, nil, errBomb, nil, errBomb, errBomb, errBomb, nil}, interrupter: true, want: errBomb},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			g, ctx := rungroup.WithContext[int, any](context.Background())

			for id, err := range tc.errs {
				id, err := id, err
				g.Go(func() (any, error) { return "", err }, tc.interrupter, id)
			}

			if err := g.Wait(); err != tc.want {
				t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
					"g.Wait() = %v; want %v",
					g, tc.errs, err, tc.want)
			}

			canceled := false
			select {
			case <-ctx.Done():
				canceled = true
			default:
			}
			if !canceled {
				t.Errorf("after %T.Go(func() error { return err }) for err in %v\n"+
					"ctx.Done() was not closed",
					g, tc.errs)
			}
		})
	}
}

func TestWithContext_GoWithFunc(t *testing.T) {
	errBomb := errors.New("group_test: fBombed in GoWithFunc")

	testCases := []struct {
		name string
		want error
		args []testArg
		n    uint64
	}{
		{
			name: "HundredRoutines_WithAllIntrptAllErr",
			want: errBomb,
			args: prepareRandomTestArgs(100, 100, 100, errBomb),
			n:    100,
		},
		{
			name: "HundredRoutines_WithNoIntrptAllErr",
			// since no interrupt hence nil error as an outcome
			want: nil,
			args: prepareRandomTestArgs(100, 0, 100, errBomb),
			n:    100,
		},
		{
			name: "HundredRoutines_WithNoIntrptNoErr",
			want: nil,
			args: prepareRandomTestArgs(100, 0, 0, errBomb),
			n:    100,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			g, ctx := rungroup.WithContext[string, any](context.Background())

			errorChan := make(chan error, tc.n)

			total := uint64(0)
			for _, testArg := range tc.args {
				testArg := testArg // https://golang.org/doc/faq#closures_and_goroutines , freaking :)
				g.GoWithFunc(func(ctx context.Context) (any, error) {
					val, err := testArg.f()
					atomic.AddUint64(&total, 1)
					return val, err
				}, ctx, testArg.interrupter, testArg.id)
			}

			if err := g.Wait(); err != tc.want {
				t.Errorf("after %T.Go() g.Wait() = %v; want %v",
					g, err, tc.want)
			}
			close(errorChan)
			if tc.want == nil && total != tc.n {
				t.Errorf("number of errors don't match with all func executed(error returned) .\n"+
					"Total Err Received: %d, Total Err Expected %d", total, tc.n)
			}
		})
	}
}

func Test(t *testing.T) {
	t.Run("Generic", func(t *testing.T) {
		g, ctx := rungroup.WithContext[int, string](context.Background())

		errBomb := errors.New("group_test: fBombed in GoWithFunc")

		errorChan := make(chan error, 3)

		for i := 0; i < 3; i++ {

			g.GoWithFunc(func(ctx context.Context) (string, error) {
				r := rand.Intn(10)
				time.Sleep(time.Duration(r) * time.Microsecond)
				return "", errBomb
			}, ctx, true, i)
		}

		if err := g.Wait(); err != errBomb {
			t.Errorf("after %T.Go() g.Wait() = %v; want %v",
				g, err, errBomb)
		}
		close(errorChan)
	})
}

func TestWithContextErrorMap(t *testing.T) {
	errorBomb := errors.New("random error bomb")
	testCases := []struct {
		name      string
		want      error
		args      []testArg
		resultMap concurrent.ConcurrentMap
	}{
		{
			name:      "HundredRoutines_WithThirtyPercentWrites_InRWMutexMap",
			want:      errorBomb,
			args:      prepareRandomTestArgs(100, 30, 30, errorBomb),
			resultMap: concurrent.NewRWMutexMap(),
		},
		{
			name:      "HundredRoutines_WithThirtyPercentWrites_InSyncMap",
			want:      errorBomb,
			args:      prepareRandomTestArgs(100, 30, 30, errorBomb),
			resultMap: new(sync.Map),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g, ctx := rungroup.WithContextResultMap[string, any](context.Background(), tc.resultMap)
			for _, testArg := range tc.args {
				g.Go(testArg.f, testArg.interrupter, testArg.id)
			}
			if err := g.Wait(); err != tc.want {
				t.Errorf("after %T.Go() g.Wait() = %v; want %v",
					g, err, tc.want)
			}

			canceled := false
			select {
			case <-ctx.Done():
				canceled = true
			default:
			}
			if !canceled {
				t.Errorf("after %T.Go()"+
					"ctx.Done() was not closed",
					g)
			}
		})
	}
}

func TestGroup_GetResultByID_GroupNilMapError(t *testing.T) {

	want := testWant{
		id:  "1",
		err: rungroup.ErrGroupNilMap,
		ok:  false,
	}
	arg := testArg{
		f: func() (any, error) {
			time.Sleep(time.Duration(3) * time.Second)
			return nil, nil
		},
		interrupter: false,
		id:          "1",
	}

	g, ctx := rungroup.WithContext[string, any](context.Background())

	ticker := time.NewTicker(50 * time.Millisecond)
	done := make(chan bool)

	go func() {
		select {
		case <-done:
			return
		case <-ticker.C:
			v, err, ok := g.GetResultByID(want.id)
			if err != want.err && ok != want.ok && v != nil {
				t.Errorf("%T.GetErrByID returned unexpected error and ok\n"+
					"Expected err: %v, Expected ok: %v"+
					"Got err: %v, Got ok: %v", g, err, ok, want.err, want.ok)
				t.Fail()
				ticker.Stop()
				done <- true
				ctx.Done()
				return
			}
		}

	}()

	g.Go(arg.f, arg.interrupter, arg.id)
	err := g.Wait()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	ticker.Stop()
	close(done)

}

func BenchmarkWithContext(b *testing.B) {
	for n := 0; n < b.N; n++ {
		errorBomb := errors.New("random error bomb")
		args := prepareRandomTestArgs(1000, 30, 30, errorBomb)
		b.ResetTimer()
		g, ctx := rungroup.WithContext[string, any](context.Background())
		for _, testArg := range args {
			g.Go(testArg.f, testArg.interrupter, testArg.id)
		}
		if err := g.Wait(); err != errorBomb {
			b.Errorf("after %T.Go() g.Wait() = %v; want %v",
				g, err, err)
		}
		<-ctx.Done()
	}
}

func BenchmarkWithContextErrorMap_RWSafeMap(b *testing.B) {

	errorBomb := errors.New("random error bomb")
	args := prepareRandomTestArgs(1000, 30, 30, errorBomb)
	b.ResetTimer()
	benchmarkWithContextErrorMap(concurrent.NewRWMutexMap(), args, errorBomb, b)

}

func BenchmarkWithContextErrorMap_SyncMap(b *testing.B) {

	var errMap sync.Map
	errorBomb := errors.New("random error bomb")
	args := prepareRandomTestArgs(1000, 30, 30, errorBomb)
	b.ResetTimer()
	benchmarkWithContextErrorMap(&errMap, args, errorBomb, b)

}

func benchmarkWithContextErrorMap(errMap concurrent.ConcurrentMap, args []testArg, errBomb error, b *testing.B) {

	for n := 0; n < b.N; n++ {
		g, ctx := rungroup.WithContextResultMap[string, any](context.Background(), errMap)
		for _, testArg := range args {
			g.Go(testArg.f, testArg.interrupter, testArg.id)
		}
		if err := g.Wait(); err != errBomb {
			b.Errorf("after %T.Go() g.Wait() = %v; want %v",
				g, err, err)
		}
		<-ctx.Done()
	}
}
