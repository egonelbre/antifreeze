package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/egonelbre/antifreeze"
)

func init() {
	antifreeze.SetFrozenLimit(1 * time.Minute)
}

var ch = make(chan int)
var x sync.Mutex

func runner() {
	antifreeze.Exclude()
	<-ch
}

func main() {
	go runner()
	http.Handle("/", http.HandlerFunc(index))
	http.Handle("/wait", http.HandlerFunc(wait))
	http.Handle("/mutex", http.HandlerFunc(mutex))
	log.Println("Make a request to either:")
	log.Println("  127.0.0.1:8000/wait")
	log.Println("  127.0.0.1:8000/mutex")
	http.ListenAndServe("127.0.0.1:8000", nil)
}

func index(w http.ResponseWriter, r *http.Request) {}

func wait(w http.ResponseWriter, r *http.Request) {
	log.Println("Receive:", r.UserAgent())
	<-ch
}

func mutex(w http.ResponseWriter, r *http.Request) {
	log.Println("Mutex:", r.UserAgent())
	x.Lock()
}
