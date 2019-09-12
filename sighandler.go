package shutdown

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hieua1/logger"
	"github.com/hieua1/observer"
)

var (
	GetSigtermHandler = getSigtermHandlerFunc()
)

type sigtermHandler struct {
	observer.Subject
	sigChannel chan os.Signal
	timeout    time.Duration
	mu         sync.Mutex
	done       chan struct{}
}

func (s *sigtermHandler) SetTimeout(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeout = duration
}

func (s *sigtermHandler) RegisterDeferFunc(f func()) {
	s.RegisterObserver(observer.Func(func(data interface{}) {
		f()
	}))
}

func (s *sigtermHandler) RegisterDeferFuncWithCancel(f func()) func() {
	obs := observer.Func(func(data interface{}) {
		f()
	})
	s.RegisterObserver(obs)
	return func() {
		s.UnregisterObserver(obs)
	}
}

func (s *sigtermHandler) WaitForSigtermHandler() {
	<-s.done
}

func getSigtermHandlerFunc() func() *sigtermHandler {
	var (
		sigtermHdl     *sigtermHandler
		sigHdlInitOnce sync.Once
	)
	return func() *sigtermHandler {
		sigHdlInitOnce.Do(func() {
			sigtermHdl = &sigtermHandler{
				Subject:    new(observer.BaseSubject),
				sigChannel: make(chan os.Signal, 1),
				timeout:    -1,
				done:       make(chan struct{}),
			}
			signal.Notify(sigtermHdl.sigChannel, os.Interrupt, syscall.SIGTERM)
			signalsReceived := 0
			go func() {
				defer close(sigtermHdl.done)
				select {
				case s := <-sigtermHdl.sigChannel:
					sigtermHdl.mu.Lock()
					defer sigtermHdl.mu.Unlock()
					signalsReceived++
					logger.S().Info("Receive signal: ", s)
					if signalsReceived == 1 {
						if sigtermHdl.timeout > 0 {
							go func() {
								select {
								case <-time.After(sigtermHdl.timeout):
									logger.S().Info("Timeout! Force application to exit.")
									os.Exit(1)
								}
							}()
						}
						logger.S().Info("Waiting for gracefully finishing current works before shutdown...")
						sigtermHdl.NotifyAll(nil)
					} else {
						logger.S().Info("One more signal received. Force application to exit.")
						os.Exit(1)
					}
				}
			}()
		})
		return sigtermHdl
	}
}
