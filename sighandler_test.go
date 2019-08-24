package shutdown

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestWaitForSigtermHandler(t *testing.T)  {
	doneFunc := false
	GetSigtermHandler().RegisterDeferFunc(func() {
		time.Sleep(1*time.Second)
		doneFunc = true
	})
	go func() {
		time.Sleep(1*time.Second)
		p, _ := os.FindProcess(syscall.Getpid())
		_ = p.Signal(os.Interrupt)
	}()
	GetSigtermHandler().WaitForSigtermHandler()
	if doneFunc != true {
		t.Fail()
	}
}
