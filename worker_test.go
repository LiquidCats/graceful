package graceful_test

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/LiquidCats/graceful/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestWorkerProcessesValues(t *testing.T) {
	ch := make(chan int)
	var processed int32
	handler := func(ctx context.Context, v int) error {
		atomic.AddInt32(&processed, 1)
		return nil
	}

	runner := graceful.Worker[int](ch, handler)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := runner(context.Background())
		assert.NoError(t, err)
	}()

	// Send values
	for i := range 5 {
		ch <- i
	}
	close(ch)

	wg.Wait()
	assert.Equal(t, int32(5), atomic.LoadInt32(&processed))
}

func TestContextDone(t *testing.T) {
	ch := make(chan int)
	handler := func(ctx context.Context, v int) error { return nil }

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	runner := graceful.Worker[int](ch, handler)
	err := runner(ctx)
	assert.EqualError(t, err, ctx.Err().Error())
}

func TestErrWorkerFailure(t *testing.T) {
	ch := make(chan int)
	handler := func(ctx context.Context, v int) error {
		return graceful.ErrWorkerFailure
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runner := graceful.Worker[int](ch, handler)
		err := runner(context.Background())
		assert.Equal(t, graceful.ErrWorkerFailure, err)
	}()

	// Send a value to trigger the error
	ch <- 42
	// Closing the channel is not strictly necessary, but does not hurt
	close(ch)

	wg.Wait()
}

func TestOtherError(t *testing.T) {
	ch := make(chan int)
	var errCount int32
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf)

	handler := func(ctx context.Context, v int) error {
		atomic.AddInt32(&errCount, 1)
		if v == 1 {
			return fmt.Errorf("test error")
		}
		return nil
	}

	runner := graceful.Worker[int](ch, handler, graceful.WithWorkerLogger(logger))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := runner(context.Background())
		assert.NoError(t, err)
	}()

	ch <- 1
	ch <- 2
	close(ch)

	wg.Wait()

	// Verify that the error was logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "runner failed")
	assert.Contains(t, logOutput, "test error")

	// Ensure that the error handler was invoked once
	assert.Equal(t, int32(2), atomic.LoadInt32(&errCount))
}

func TestWithWorkerLogger(t *testing.T) {
	ch := make(chan int)
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf)

	handler := func(ctx context.Context, v int) error { return nil }

	runner := graceful.Worker[int](ch, handler, graceful.WithWorkerLogger(logger))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := runner(context.Background())
		assert.NoError(t, err)
	}()

	// No values; close channel immediately to trigger "channel closed" log
	close(ch)

	wg.Wait()

	// Verify that the logger emitted the "channel closed" message
	logOutput := buf.String()
	assert.Contains(t, logOutput, "channel closed")
}
