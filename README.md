# antifreeze

antifreeze is a package that detects goroutines that have been waiting for too long.

**Warning: this package is alpha and may not work as intended.**

You can exclude functions that may block forever with `antifreeze.Exclude` and `antifreeze.ExcludeNamed`.

Example program that will panic ~1min after you have made a request to it.

``` go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/egonelbre/antifreeze"
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
	fmt.Println("R:", r)
	<-ch
}
```

See [godoc](https://godoc.org/github.com/egonelbre/antifreeze) for more information.