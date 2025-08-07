package pool

type Result[O any] struct {
	order  uint64
	result O
	err    error
}

func (r Result[O]) Error() error {
	return r.err
}

func (r Result[O]) Result() O {
	return r.result
}

type inputData[I any] struct {
	order uint64
	data  I
}
