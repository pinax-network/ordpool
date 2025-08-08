package ordpool

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNotRunning = errors.New("pool is not running")
)

type WorkerFunc[I, O any] func(in I) (out O, err error)

type WorkerPool[I, O any] struct {
	inChan      chan inputData[I] // inChan is the channel we are writing the data into that we get when Process is called
	aggChan     chan *Result[O]   // aggChan is the channel each worker is writing the result of the WorkerFunc into
	outChan     chan *Result[O]   // outChan is the channel holding the data from the aggChan in an ordered way
	globalOrder uint64            // globalOrder is a global counter for the order
	workers     []*worker[I, O]
	workerWg    sync.WaitGroup
	resultHeap  *resultHeap[O]
	isRunning   bool
}

// NewPool creates a new, ordered worker pool that allows processing continuous streams of data while preserving the
// order of events.
// Results are written to the output channel returned. The channel will be closed after Shutdown has been called and all
// remaining inputs have been processed.
func NewPool[I, O any](numWorkers int, workerFunc WorkerFunc[I, O]) (pool *WorkerPool[I, O], outputChan <-chan *Result[O]) {
	pool = &WorkerPool[I, O]{
		inChan:     make(chan inputData[I]),
		aggChan:    make(chan *Result[O]),
		outChan:    make(chan *Result[O]),
		workers:    make([]*worker[I, O], 0, numWorkers),
		workerWg:   sync.WaitGroup{},
		resultHeap: &resultHeap[O]{},
	}

	for i := 0; i < numWorkers; i++ {
		wrk := newWorker[I, O](i, pool.inChan, pool.aggChan, workerFunc)
		pool.workers = append(pool.workers, wrk)
		pool.workerWg.Add(1)
		go func(wrk *worker[I, O]) {
			defer pool.workerWg.Done()
			wrk.run()
		}(wrk)
	}

	go pool.orderOutput()
	pool.isRunning = true

	return pool, pool.outChan
}

// Process adds the given input to the pool to be processed by a worker. When done, the Result is pushed to the output
// channel returned by NewPool in an ordered way.
// Returns the global order for the input. In case Shutdown has already been called, an ErrNotRunning is returned.
func (w *WorkerPool[I, O]) Process(in I) (uint64, error) {
	if !w.isRunning {
		return 0, ErrNotRunning
	}

	w.globalOrder++
	w.inChan <- inputData[I]{order: w.globalOrder, data: in}

	return w.globalOrder, nil
}

// Shutdown gracefully shuts down the pool. It signals all workers to stop after finishing the input queue. When all
// results are written, the output channel is closed.
// Note that this is not blocking. The caller should wait for the output channel to be closed.
func (w *WorkerPool[I, O]) Shutdown() {
	if !w.isRunning {
		return
	}

	w.isRunning = false
	close(w.inChan)

	go func() {
		w.workerWg.Wait()
		close(w.aggChan)
	}()
}

// orderOutput listens on the aggChan for results from the workers and writes them into the outChan in order.
func (w *WorkerPool[I, O]) orderOutput() error {
	currentOrder := uint64(1)
	for {
		select {
		case res, ok := <-w.aggChan:
			if ok {
				heap.Push(w.resultHeap, res)
			}

			// Peek checks the heap element with the minimal order. If this matches the current order, then we found the
			// next element in order, remove it from the heap using Pop and write it into the outChan.
			for top, ok := w.resultHeap.Peek(); ok && top.order == currentOrder; top, ok = w.resultHeap.Peek() {
				w.outChan <- heap.Pop(w.resultHeap).(*Result[O])
				currentOrder++
			}

			// we should have drained all the elements from the heap
			if !ok {
				close(w.outChan)
				if w.resultHeap.Len() > 0 {
					return fmt.Errorf("failed to drain heap, %d elements left", w.resultHeap.Len())
				}
				return nil
			}
		}
	}
}
