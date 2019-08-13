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

type SigtermHandler interface {
	RegisterDeferFunc(func())
	SetTimeout(time.Duration)
}

type sigtermHandler struct {
	observer.Subject
	sigChannel chan os.Signal
	timeout    time.Duration
}

func (s *sigtermHandler) SetTimeout(duration time.Duration) {
	s.timeout = duration
}

func (s *sigtermHandler) RegisterDeferFunc(f func()) {
	s.RegisterObserver(observer.Func(func(data interface{}) {
		f()
	}))
}

func getSigtermHandlerFunc() func() SigtermHandler {
	var (
		sigtermHdl     *sigtermHandler
		sigHdlInitOnce sync.Once
	)
	return func() SigtermHandler {
		sigHdlInitOnce.Do(func() {
			sigtermHdl = &sigtermHandler{
				Subject:    new(observer.BaseSubject),
				sigChannel: make(chan os.Signal, 1),
				timeout:    -1,
			}
			signal.Notify(sigtermHdl.sigChannel, os.Interrupt, syscall.SIGTERM)
			signalsReceived := 0
			go func() {
				select {
				case s := <-sigtermHdl.sigChannel:
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
