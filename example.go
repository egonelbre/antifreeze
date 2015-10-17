// +build ignore

package main

import (
	"net/http"

	"github.com/egonelbre/antifreeze"

	_ "net/http/pprof"
)

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
