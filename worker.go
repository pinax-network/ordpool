package ordpool

type worker[I, O any] struct {
	id            int
	inputChan     chan inputData[I]
	aggregateChan chan *Result[O]
	workerFunc    WorkerFunc[I, O]
	stopChan      chan bool
}

func newWorker[I, O any](id int, inputChan chan inputData[I], aggregateChan chan *Result[O], workerFunc WorkerFunc[I, O]) *worker[I, O] {
	return &worker[I, O]{
		id:            id,
		inputChan:     inputChan,
		aggregateChan: aggregateChan,
		workerFunc:    workerFunc,
		stopChan:      make(chan bool),
	}
}

func (w *worker[I, O]) run() {
	for {
		select {
		case <-w.stopChan:
			return
		case in, ok := <-w.inputChan:
			if !ok {
				return
			}
			res, err := w.workerFunc(in.data)
			w.aggregateChan <- &Result[O]{order: in.order, result: res, err: err}
		}
	}
}

func (w *worker[I, O]) shutdown() {
	w.stopChan <- true
}
