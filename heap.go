package ordpool

type resultHeap[O any] []*Result[O]

func (h resultHeap[O]) Len() int {
	return len(h)
}

func (h resultHeap[O]) Less(i, j int) bool {
	return h[i].order < h[j].order
}

func (h resultHeap[O]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *resultHeap[O]) Push(x interface{}) {
	*h = append(*h, x.(*Result[O]))
}

func (h resultHeap[O]) Peek() (*Result[O], bool) {
	if len(h) > 0 {
		return h[0], true
	}
	return nil, false
}

func (h *resultHeap[O]) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
