package goo

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

func GracefulExit() {
	// a shitty way to see if exitCtx has been initialized by DI
	if exitCtx == nil {
		// DI does not require graceful exit
		return
	}

	exitCtx.doExit()
}

type ShutdownContext struct {
	context.Context

	exitFns [](func() error)

	mu        sync.Mutex
	wg        sync.WaitGroup
	waitCount int64
	logger    *slog.Logger
}

func (c *ShutdownContext) doExit() {
	// may be called via GracefulExit or sigint. Lock this so there is only one
	// caller, and blocking everyone until exit.
	c.mu.Lock()

	// wait for blocking code
	c.waitBlocks()

	// run exit cleanups
	c.runExitFns()

	os.Exit(0)
}

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

var ErrShutdown = errors.New("process is shutting down")

// BlockExit runs a function and wait for it before shutting down a process
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

func (c *ShutdownContext) OnExit(fn func() error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.exitFns = append(c.exitFns, fn)
}

var exitCtx *ShutdownContext
var exitCtxOnce sync.Once

func ProvideShutdownContext(log *slog.Logger) (*ShutdownContext, error) {
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
			<-exitCtx.Done()
			exitCtx.doExit()
		}()
	})

	return exitCtx, nil
}
