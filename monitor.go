package antifreeze

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultBufferSize specifies how much buffer to use for reading stack
	// profiles by default.
	DefaultBufferSize = 8 << 20
	// DefaultFrozenLimit specifies when a goroutine should be considered stuck.
	DefaultFrozenLimit = 10 * time.Minute
)

var (
	mu         sync.Mutex
	_limit     = int(DefaultFrozenLimit / time.Minute)
	_buffer    = make([]byte, DefaultBufferSize)
	_whitelist = make(map[string]struct{})
)

var faulting = map[string]struct{}{
	"chan send":    {},
	"chan receive": {},
}

// SetBufferSize sets the maximum buffer size for reading stack profiles.
// Set this lower, e.g. 1<<10 (10MB) to reduce memory footprint at the cost of
// possibly not tracking some goroutines.
// Set it higher, e.g. 64<<10 (64MB) if you expect a lot of goroutines.
func SetBufferSize(size int) {
	mu.Lock()
	_buffer = make([]byte, size)
	mu.Unlock()
}

// SetFrozenLimit sets the limit after which the goroutine should be considered
// as frozen.
func SetFrozenLimit(limit time.Duration) {
	mu.Lock()
	if limit < time.Minute {
		panic("limit must be larger than 1 minute")
	}
	_limit = int(limit / time.Minute)
	mu.Unlock()
}

// Exclude excludes all goroutines that contain current function in the callstack.
func Exclude() {
	if pc, _, _, ok := runtime.Caller(1); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			ExcludeNamed(fn.Name())
		}
	}
}

// ExcludeNamed excludes all goroutines that contain the named func
// The func name must be fully qualified.
//
//     antifreeze.ExcludeNamed("main.runner")
//     antifreeze.ExcludeNamed("net/http.ListenAndServe")
//     antifreeze.ExcludeNamed("net.(*pollDesc).Wait")
func ExcludeNamed(name string) {
	mu.Lock()
	next := make(map[string]struct{}, len(_whitelist)+1)
	for fnname := range _whitelist {
		next[fnname] = struct{}{}
	}
	next[name] = struct{}{}
	_whitelist = next
	mu.Unlock()
}

func params() (buffer []byte, limit int, whitelist map[string]struct{}) {
	mu.Lock()
	buffer = _buffer
	limit = _limit
	whitelist = _whitelist
	mu.Unlock()
	return
}

func init() {
	go monitor()
}

func monitor() {
	check()
	for range time.Tick(30 * time.Second) {
		check()
	}
}

func skiptrace(scanner *bufio.Scanner) {
	for scanner.Scan() && scanner.Text() == "" {
		return
	}
}

func isfaulting(kind string) bool {
	return kind == "chan send" || kind == "chan receive"
}

func check() {
	buf, limit, whitelist := params()
	n := runtime.Stack(buf, true)
	scanner := bufio.NewScanner(bytes.NewReader(buf[:n]))

	for scanner.Scan() {
		// goroutine 60 [chan receive, 1 minutes]:
		line := scanner.Text()
		if line == "" {
			continue
		}

		// has it been stuck at least 1 minute
		if !strings.Contains(line, "minutes") {
			skiptrace(scanner)
			continue
		}

		rest := line

		// strip "goroutine "
		rest = rest[10:]
		p := strings.IndexByte(rest, ' ')

		// extract id
		id := rest[:p]
		rest = rest[p+1:]

		p = strings.IndexByte(rest, ',')
		kind := rest[1:p]
		rest = rest[p+2:]

		if !isfaulting(kind) {
			skiptrace(scanner)
			continue
		}

		p = strings.IndexByte(rest, ' ')
		minutes_str := rest[:p]
		minutes, err := strconv.Atoi(minutes_str)
		if err != nil {
			skiptrace(scanner)
			continue
		}

		if minutes >= limit {
			whitelisted := false
			var outbuf bytes.Buffer
			fmt.Fprintln(&outbuf, line)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Fprintln(&outbuf, line)
				if line == "" {
					break
				}
				p := strings.LastIndexByte(line, '(')
				if p >= 0 {
					if _, exists := whitelist[line[:p]]; exists {
						whitelisted = true
						skiptrace(scanner)
						break
					}
				}
			}

			if !whitelisted {
				os.Stdout.Write(outbuf.Bytes())
				panic("goroutine " + id + " was frozen")
			}
		}

		skiptrace(scanner)
	}
}
