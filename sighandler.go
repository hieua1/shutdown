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

type SigtermHandler interface {
	RegisterDeferFunc(func())
}

type sigtermHandler struct {
	observer.Subject
	sigChannel chan os.Signal
	timeout    time.Duration
}

var (
	GetSigtermHandler = getSigtermHandlerFunc()
)

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
			go func() {
				select {
				case s := <-sigtermHdl.sigChannel:
					logger.S().Info("Receive signal: ", s)
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
				}
			}()
		})
		return sigtermHdl
	}
}

func (s *sigtermHandler) RegisterDeferFunc(f func()) {
	s.RegisterObserver(observer.Func(func(data interface{}) {
		f()
	}))
}
