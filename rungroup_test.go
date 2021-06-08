package rungroup_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/bharat-rajani/rungroup"
)

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
		{name: "ErrorBombWhenAnyIntrpt", errs: []error{errBomb, nil}, interrupter: true, want: errBomb},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			g, ctx := rungroup.WithContext(context.Background())

			for id, err := range tc.errs {
				id, err := id, err
				g.Go(func() error { return err }, tc.interrupter, strconv.Itoa(id))
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
