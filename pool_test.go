package ordpool

import (
	"math/rand"
	"slices"
	"sync"
	"testing"
	"time"
)

func Test_OrderedOutput(t *testing.T) {

	workerFunc := func(in int) (int, error) {
		sleep := time.Duration((rand.Int31n(10)+1)*100) * time.Millisecond
		time.Sleep(sleep)
		return in, nil
	}

	pool, outChan := NewPool(10, workerFunc)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go checkOutput(t, &wg, outChan, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	for i := 1; i <= 10; i++ {
		_, err := pool.Process(i)
		if err != nil {
			t.Error(err)
		}
	}

	pool.Shutdown()
	wg.Wait()
}

func checkOutput(t *testing.T, wg *sync.WaitGroup, outChan <-chan *Result[int], expectedRes []int) {

	actualRes := make([]int, 0, len(expectedRes))
	for res := range outChan {
		if res.Error() != nil {
			t.Error(res.err)
		}
		actualRes = append(actualRes, res.Result())
	}
	if len(actualRes) != len(expectedRes) {
		t.Errorf("expected %d results, got %d", len(expectedRes), len(actualRes))
	}
	if !slices.Equal(actualRes, expectedRes) {
		t.Errorf("expected %v, got %v", expectedRes, actualRes)
	}

	wg.Done()
}
