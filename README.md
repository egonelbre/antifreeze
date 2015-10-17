# antifreeze
antifreeze is a package that detects goroutines that have been waiting for too long.

You can exclude functions that may block forever with `antifreeze.Exclude` and `antifreeze.ExcludeNamed`.

Example program that will panic after 1min, but only if you have made a request to it.

``` go
package main

import (
	"net/http"
	"time"

	"github.com/egonelbre/antifreeze"

	_ "net/http/pprof"
)

func init() {
	antifreeze.SetFrozenLimit(1 * time.Minute)
}

var ch = make(chan int)

func runner() {
	antifreeze.Exclude()
	<-ch
}

func main() {
	go runner()
	http.Handle("/", http.HandlerFunc(index))
	http.ListenAndServe(":8000", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	<-ch
}
```

See [godoc](https://godoc.org/github.com/egonelbre/antifreeze) for more information.