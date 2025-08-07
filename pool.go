package pool

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
	aggChan     chan Result[O]    // aggChan is the channel each worker is writing the result of the WorkerFunc into
	outChan     chan *Result[O]   // outChan is the channel holding the data from the aggChan in an ordered way
	globalOrder uint64            // globalOrder is a global counter for the order
	workers     []*worker[I, O]
	workerWg    sync.WaitGroup
	resultHeap  *resultHeap[O]
	isRunning   bool
}

func NewPool[I, O any](numWorkers int, workerFunc WorkerFunc[I, O]) (*WorkerPool[I, O], <-chan *Result[O]) {
	res := &WorkerPool[I, O]{
		inChan:     make(chan inputData[I]),
		aggChan:    make(chan Result[O]),
		outChan:    make(chan *Result[O]),
		workers:    make([]*worker[I, O], 0, numWorkers),
		workerWg:   sync.WaitGroup{},
		resultHeap: &resultHeap[O]{},
	}

	for i := 0; i < numWorkers; i++ {
		wrk := newWorker[I, O](i, res.inChan, res.aggChan, workerFunc)
		res.workers = append(res.workers, wrk)
		res.workerWg.Add(1)
		go func(wrk *worker[I, O]) {
			defer res.workerWg.Done()
			wrk.Run()
		}(wrk)
	}

	go res.orderOutput()
	res.isRunning = true

	return res, res.outChan
}

func (w *WorkerPool[I, O]) Process(in I) (uint64, error) {
	if !w.isRunning {
		return 0, ErrNotRunning
	}

	w.globalOrder++
	w.inChan <- inputData[I]{order: w.globalOrder, data: in}

	return w.globalOrder, nil
}

func (w *WorkerPool[I, O]) Shutdown() {
	w.isRunning = false
	close(w.inChan)

	go func() {
		w.workerWg.Wait()
		close(w.aggChan)
	}()

	//for _, wrk := range w.workers {
	//	wrk.Shutdown()
	//}
	//close(w.outChan)
}

// orderOutput listens on the aggChan for results from the workers and writes them into the outChan in order.
func (w *WorkerPool[I, O]) orderOutput() error {
	currentOrder := uint64(1)
	for {
		select {
		case res, ok := <-w.aggChan:
			if ok {
				heap.Push(w.resultHeap, &res)
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
