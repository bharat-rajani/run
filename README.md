# rungroup
[![GoDoc](https://pkg.go.dev/badge/github.com/bharat-rajani/rungroup)](https://godoc.org/github.com/bharat-rajani/rungroup)
[![GoVersion](https://img.shields.io/github/go-mod/go-version/bharat-rajani/rungroup)](https://github.com/bharat-rajani/rungroup/blob/main/go.mod)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg)](https://goreportcard.com/report/github.com/bharat-rajani/rungroup)
[![MIT licensed](https://img.shields.io/github/license/bharat-rajani/rungroup)](https://github.com/bharat-rajani/rungroup/blob/main/LICENSE)

### Goroutines lifecycle manager

Rungroup was created to manage multiple goroutines which may or may not interrupt other goroutines on error.

Rationale:
Whilst hacking with golang, some or the other day you will encounter a situation where you have to manage multiple goroutines which can interrupt other goroutines if an error occurs.

A rungroup is essentially a composition of:
- goroutines
- optional concurrent(thread safe) ErrorMap to track errors from different goroutines
- context cancel func, when cancelled can stop other goroutines

A goroutine in rungroup is essentially composition of:
- a user(programmer) defined function which returns error
- an identifier (string) which may help you to track goroutine error.

> Rungroup is inspired by errorgroup.

### Installation:
```shell
go get -u github.com/bharat-rajani/rungroup   
```

### Example:

#### A quick and simple example, where we need to call 3 REST Endpoints concurrently.

Three gorutines:
- F_API, interrupter
- S_API
- T_API, interrupter

Let's say we don't care about the response from second REST API Endpoint, hence that routine cannot interrupt other routines (F_API, T_API).
Now as soon as there is an error in F_API (or in T_API) then all other goroutines will be stopped.

```go
package main

import (
	"fmt"
	"context"
	"net/http"
	"github.com/bharat-rajani/rungroup"
	"github.com/bharat-rajani/rungroup/pkg/concurrent"
)

func main() {
	g, ctx := rungroup.WithContextErrorMap(context.Background(),concurrent.NewRWMutexMap())
	// g, ctx := rungroup.WithContextErrorMap(context.Background(),new(sync)) //refer Benchmarks for performance difference
	
	var fResp, sResp, tResp *http.Response
	g.GoWithFunc(func(ctx context.Context) error {
		// error placeholder
		var tErr error
		fResp, tErr = http.Get("F_API_URL")
		if tErr != nil {
			return tErr
		}
		return nil
	}, ctx, true, "F_API")

	g.GoWithFunc(func(ctx context.Context) error {
		// error placeholder
		var tErr error
		sResp, tErr = http.Get("S_API_URL")
		if tErr != nil {
			return tErr
		}
		return nil

	}, ctx, false, "S_API")

	g.GoWithFunc(func(ctx context.Context) error {
		// error placeholder
		var tErr error
		tResp, tErr = http.Get("T_API_URL")
		if tErr != nil {
			return tErr
		}
		return nil
	}, ctx, true, "T_API")
    
	// returns first error from interrupter routine
	err := g.Wait()
	if err != nil {
		fmt.Println(err)
	}
}
```


#### What if error occurs in "S_API" routine ? How can I retrieve its error?

Since "S_API" is a non interrupter goroutine hence the only way to track its error is by:

```golang
err, ok := g.GetErrorByID("S_API")
if ok && err!=nil{
	fmt.Println(err)
}
```
 
#### I don't want to concurrently Read or Write errors.

Ok, I heard you, using concurrent maps comes with performance tradeoff.
If you don't want to track errors of all gorutines and you are happy with first occurring error, then just use rungroup WithContext:

```golang
g, ctx := rungroup.WithContext(context.Background(),concurrent.NewRWMutexMap())
...
...
```

> Note: When you use rungroup.WithContext (no error tracking) then calling g.GetErrorByID() will yield you a nice uninitialized map error and ok = false.