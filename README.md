# ordpool

ordpool is a generic, ordered worker pool that accepts a continuous stream of input, processes it concurrently and
returns the results in the same order as the input.

## Example usage

```go
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/pinax-network/ordpool"
)

func main() {
	pool, outChan := ordpool.NewPool(10, workerFunc)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go handleResult(wg, outChan)

	for i := 1; i <= 10; i++ {
		_, _ = pool.Process(i)
	}

	pool.Shutdown() // closes the input channel and signals the workers to stop after finishing the input queue
	wg.Wait()       // wait until the output channel is drained
}

func handleResult(wg *sync.WaitGroup, outChan <-chan *ordpool.Result[string]) {
	for {
		select {
		case res, ok := <-outChan:
			if !ok { // the output channel has been closed, so we are done here
				wg.Done()
				return
			}
			fmt.Println(res.Result())
		}
	}
}

// the worker function sleeps randomly for 100-900 ms to simulate long-running tasks
func workerFunc(in int) (string, error) {
	sleep := time.Duration((rand.Int31n(9)+1)*100) * time.Millisecond
	time.Sleep(sleep)
	return fmt.Sprintf("input %d processed within %d ms", in, sleep.Milliseconds()), nil
}
```

The workerFunc will be executed in parallel on 10 workers, but the output channel will return the results ordered:
```
# go run .
input 1 processed within 200 ms
input 2 processed within 700 ms
input 3 processed within 200 ms
input 4 processed within 200 ms
input 5 processed within 300 ms
input 6 processed within 900 ms
input 7 processed within 300 ms
input 8 processed within 800 ms
input 9 processed within 800 ms
input 10 processed within 400 ms
```