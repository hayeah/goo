package goo

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// setupTest resets all the relevant global variables, and then uses t.Cleanup
// to restore them after each test. You can also set up any default mocks here.
func setupTest(t *testing.T) {
	// Capture the original values
	origExitFunc := exitFunc
	origSignalFunc := signalFunc
	origResetFunc := resetFunc
	origExitCtx := exitCtx
	origExitCtxOnce := exitCtxOnce
	origUsingShutdownContext := usingShutdownContext

	// Provide default mocks here if desired
	exitFunc = func(code int) {
		// By default, do nothing, or capture the code in a local variable if you want.
	}
	signalFunc = func(c chan<- os.Signal, _ ...os.Signal) {
		// By default, do nothing unless the test configures it further
	}
	resetFunc = func(_ ...os.Signal) {
		// By default, do nothing
	}
	exitCtx = nil
	exitCtxOnce = sync.Once{}
	usingShutdownContext = false

	// Now ensure these are restored after the test ends
	t.Cleanup(func() {
		exitFunc = origExitFunc
		signalFunc = origSignalFunc
		resetFunc = origResetFunc
		exitCtx = origExitCtx
		exitCtxOnce = origExitCtxOnce
		usingShutdownContext = origUsingShutdownContext
	})
}

func TestProvideShutdownContext_NotUsed(t *testing.T) {
	assert := assert.New(t)

	setupTest(t) // <--- ensures we start from a known baseline

	// ProvideShutdownContext should fail if usingShutdownContext == false.
	usingShutdownContext = false
	ctx, err := ProvideShutdownContext(slog.Default())

	assert.Nil(ctx, "Context should be nil when not usingShutdownContext")
	assert.Error(err, "Expected error when ProvideShutdownContext is called with usingShutdownContext=false")
}

func TestProvideShutdownContext_Success(t *testing.T) {
	assert := assert.New(t)

	setupTest(t)

	usingShutdownContext = true // let ProvideShutdownContext do its full initialization
	ctx, err := ProvideShutdownContext(slog.Default())

	assert.NotNil(ctx, "Should have a valid context now")
	assert.NoError(err)
}

func TestShutdownContext_OnExit(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	sc := &ShutdownContext{}
	sc.logger = slog.Default()

	var callCount int32
	sc.OnExit(func() error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})
	sc.OnExit(func() error {
		atomic.AddInt32(&callCount, 1)
		return nil
	})

	// This method is private, but we can call it directly since we’re in package goo.
	sc.runExitFns()
	assert.Equal(int32(2), callCount)
}

func TestShutdownContext_BlockExit_Success(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	sc := &ShutdownContext{
		Context: context.Background(),
		logger:  slog.Default(),
	}

	var wg sync.WaitGroup
	wg.Add(1)

	err := sc.BlockExit(func() error {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	assert.NoError(err)
	wg.Wait()

	// Make sure the waitCount is back to zero
	assert.Equal(int64(0), sc.waitCount)
}

func TestShutdownContext_BlockExit_AlreadyCanceled(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	sc := &ShutdownContext{
		Context: ctx,
		logger:  slog.Default(),
	}

	err := sc.BlockExit(func() error {
		t.Fatal("this block should never be executed if context is canceled")
		return nil
	})

	assert.ErrorIs(err, ErrShutdown)
}

func TestShutdownContext_DoExit_CallsExitFunc(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	// We'll override exitFunc for this test, capturing the exit code
	var capturedCode int
	exitFunc = func(code int) {
		capturedCode = code
	}

	sc := &ShutdownContext{logger: slog.Default()}
	sc.doExit(42)
	assert.Equal(42, capturedCode)
}

func TestGracefulExit_IfNoExitCtxDoesNothing(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	var capturedCode int
	exitFunc = func(code int) {
		capturedCode = code
	}

	// If exitCtx is nil, gracefulExit shouldn't do anything
	exitCtx = nil
	gracefulExit(99)
	assert.Equal(0, capturedCode, "Should remain 0 if exitCtx is nil")
}

func TestGracefulExit_WithExitCtxCallsDoExit(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	var capturedCode int
	exitFunc = func(code int) {
		capturedCode = code
	}

	sc := &ShutdownContext{logger: slog.Default()}
	exitCtx = sc

	gracefulExit(123)
	assert.Equal(123, capturedCode)
}

func TestProvideShutdownContext_SignalHandling(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	usingShutdownContext = true

	// We'll intercept signals by providing a channel we control.
	sigChan := make(chan os.Signal, 3)
	// Mock the signal function to store our channel for later use
	var storedChan chan<- os.Signal
	signalFunc = func(c chan<- os.Signal, _ ...os.Signal) {
		// Store the channel so we can send signals to it
		storedChan = c
		// Also forward to our test channel so we can control signals
		go func() {
			for sig := range sigChan {
				storedChan <- sig
			}
		}()
	}

	var resetCalled bool
	resetFunc = func(_ ...os.Signal) {
		resetCalled = true
	}

	ctx, err := ProvideShutdownContext(slog.Default())
	assert.NotNil(ctx)
	assert.NoError(err)

	// Simulate 3 signals
	sigChan <- os.Interrupt
	sigChan <- os.Interrupt
	sigChan <- os.Interrupt

	// The third interrupt should trigger resetFunc
	time.Sleep(5 * time.Millisecond) // small delay for the goroutine to process
	assert.True(resetCalled, "resetFunc should have been called after the 3rd signal")
}

func TestMain_Success(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	usingShutdownContext = true

	var exitCode int
	exitFunc = func(c int) {
		exitCode = c
	}

	initFunc := func() (Runner, error) {
		return &mockRunner{}, nil
	}

	Main(initFunc)
	assert.Equal(0, exitCode, "Expect exit code 0 on success")
}

func TestMain_InitError(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	usingShutdownContext = true

	var exitCode int
	exitFunc = func(c int) {
		exitCode = c
	}

	initFunc := func() (Runner, error) {
		return nil, errors.New("some init error")
	}

	Main(initFunc)
	assert.Equal(1, exitCode, "Expect exit code 1 on init error")
}

func TestMain_RunError(t *testing.T) {
	assert := assert.New(t)
	setupTest(t)

	usingShutdownContext = true

	var exitCode int
	exitFunc = func(c int) {
		exitCode = c
	}

	initFunc := func() (Runner, error) {
		return &mockRunner{failOnRun: true}, nil
	}

	Main(initFunc)
	assert.Equal(1, exitCode, "Expect exit code 1 on run error")
}

// mockRunner is just a test stub for Runner
type mockRunner struct {
	failOnRun bool
}

func (m *mockRunner) Run() error {
	if m.failOnRun {
		return errors.New("run error")
	}
	return nil
}
