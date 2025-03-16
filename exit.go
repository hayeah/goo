// Package goo provides utilities for application lifecycle management and dependency injection.
//
// The package implements graceful shutdown handling, allowing applications to properly
// clean up resources and complete ongoing operations before terminating.
package goo

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ShutdownContext extends context.Context to manage application shutdown.
// It provides mechanisms for graceful termination, including waiting for
// ongoing operations to complete and executing cleanup functions.
type ShutdownContext struct {
	context.Context

	exitFns [](func() error) // Functions to execute during shutdown

	mu        sync.Mutex     // Protects concurrent access to shutdown operations
	wg        sync.WaitGroup // Tracks ongoing operations that must complete before shutdown
	waitCount int64          // Counter for active blocking operations
	logger    *slog.Logger   // Logger for shutdown-related messages
}

// doExit performs the actual shutdown sequence.
// It may be called via GracefulExit or sigint. Lock this so there is only one
// caller, and blocking everyone until exit.
func (c *ShutdownContext) doExit(code int) {
	// may be called via GracefulExit or sigint. Lock this so there is only one
	// caller, and blocking everyone until exit.
	c.mu.Lock()

	// wait for blocking code
	c.waitBlocks()

	// run exit cleanups
	c.runExitFns()

	os.Exit(code)
}

// waitBlocks waits for all blocking operations to complete before shutdown.
// It periodically logs progress if waiting takes longer than expected.
func (c *ShutdownContext) waitBlocks() {
	// c.log.Debug().Msg("waiting for exit blocks")
	log := c.logger

	// if exit blocks take longer than 5 seconds, start logging progress
	go func() {
		for {
			count := atomic.LoadInt64(&c.waitCount)
			if count == 0 {
				return
			}

			log.Debug("waiting for exit blocks", "count", count)

			time.Sleep(3 * time.Second)
		}
	}()

	c.wg.Wait()
}

// runExitFns executes all registered cleanup functions.
// Errors from individual functions are logged but do not stop the execution
// of subsequent functions.
func (c *ShutdownContext) runExitFns() {
	log := c.logger

	if len(c.exitFns) > 0 {
		log.Debug("running exit functions", "count", len(c.exitFns))
	}

	for _, fn := range c.exitFns {
		err := fn()

		if err != nil {
			log.Debug("exit function error", "error", err.Error())
		}

	}

	// err := slice.EachParallel(c.exitFns, func(fn func() error) error {
	// 	return fn()
	// }, 16)

	// if err != nil {
	// 	c.log.Debug().Err(err).Msg("exit function errors")
	// }
}

// ErrShutdown is returned when an operation is attempted during shutdown.
var ErrShutdown = errors.New("process is shutting down")

// BlockExit runs a function and waits for it to complete before shutting down the process.
// If the process is already shutting down, it returns ErrShutdown without executing the function.
// This is useful for operations that must complete before the application terminates.
func (c *ShutdownContext) BlockExit(fn func() error) error {
	// return error if process is already shutting down
	select {
	case <-c.Done():
		return ErrShutdown
	default:
	}

	c.wg.Add(1)
	atomic.AddInt64(&c.waitCount, 1)
	err := fn()
	atomic.AddInt64(&c.waitCount, -1)
	c.wg.Done()
	return err
}

// OnExit registers a function to be executed during shutdown.
// These functions are run after all blocking operations have completed.
// Cleanup operations like closing database connections or releasing resources
// should be registered using this method.
func (c *ShutdownContext) OnExit(fn func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.exitFns = append(c.exitFns, fn)
}

// Global singleton instance of ShutdownContext
var exitCtx *ShutdownContext
var exitCtxOnce sync.Once
var usingShutdownContext bool

func gracefulExit(code int) {
	// If no dependency requires ShutdownContext, then gracefulExit just do nothing.
	if exitCtx == nil {
		return
	}

	exitCtx.doExit(code)
}

// ProvideShutdownContext creates and returns a ShutdownContext for dependency injection.
//
// This function is typically used with dependency injection frameworks like Wire.
// It ensures that the application properly handles shutdown signals and manages
// cleanup operations. The returned ShutdownContext can be used to register
// cleanup functions and block operations during shutdown.
//
// Note: This function will return an error if the application was not started
// using goo.Main(). Always use goo.Main() as the entry point for your application
// to ensure proper shutdown handling, even if your application doesn't explicitly
// use the ShutdownContext.
//
// Example usage with Wire:
//
//	func ProvideApp(ctx *goo.ShutdownContext, db *sql.DB) *App {
//	    app := &App{db: db}
//	    ctx.OnExit(func() error {
//	        return db.Close()
//	    })
//	    return app
//	}
func ProvideShutdownContext(log *slog.Logger) (*ShutdownContext, error) {
	// ensures that the library user has used goo.Main to ensure graceful shutdown
	if !usingShutdownContext {
		return nil, errors.New("ShutdownContext not enabled")
	}

	// enforce that exitCtx is initialized once
	exitCtxOnce.Do(func() {
		bg := context.Background()

		sigs := make(chan os.Signal, 32)
		signal.Notify(sigs, os.Interrupt)

		ctx, cancel := context.WithCancel(bg)

		exitCtx = &ShutdownContext{Context: ctx, logger: log}

		// 3 sigints to force an immediate exit
		i := 0
		go func() {
			for {
				<-sigs

				if i == 0 {
					cancel()
				}

				log.Debug("graceful exit. 3 sigints to exit immediately", "countdown", 3-i)

				i++

				if i == 3 {
					// cancel the handler. next sigint will force an exit
					signal.Reset(os.Interrupt)
				}
			}
		}()

		go func() {
			// Wait for context cancellation by sigint
			<-exitCtx.Done()
			// On Unix-like systems, when a process is terminated by a signal, its exit code is set to 128 plus that signal's number.
			exitCtx.doExit(128 + int(syscall.SIGINT))
		}()
	})

	return exitCtx, nil
}

type Runner interface {
	Run() error
}

// Main is a wrapper to initialize the shutdown context and run the application.
//
// Main serves as the primary entry point for applications using the goo package.
// It initializes the application, runs it, and ensures graceful shutdown when
// the application terminates or receives interrupt signals.
//
// The init function is typically a dependency injection function (like a Wire-generated
// InitializeApp function) that creates and configures the application's components.
//
// Important: Always use goo.Main() as the entry point for your application, even if
// your application doesn't explicitly use ShutdownContext. This ensures proper
// signal handling and graceful shutdown capabilities are available if needed by
// any part of your application or its dependencies.
//
// Example:
//
//	func main() {
//	    goo.Main(app.InitializeApp)
//	}
//
// Where InitializeApp is a function that returns a Runner implementation.
func Main[T Runner](init func() (T, error)) {
	// the init function is a wire injection.
	//
	// if any wire provider requires the shutdown context, we need to ensure that gracefulExit is called.

	usingShutdownContext = true

	runner, err := init()
	if err != nil {
		log.Println("init", err)
		gracefulExit(1)
	}

	err = runner.Run()
	if err != nil {
		log.Println("run", err)
		gracefulExit(1)
	}

	gracefulExit(0)
}
