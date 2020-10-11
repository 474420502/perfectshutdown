package perfectshutdown

import (
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

// PerfectShutdown 完美关闭程序
type PerfectShutdown struct {
	loop int32

	beforeparams interface{}
	before       func(params interface{})
}

// New 创建完美关闭程序
func New() *PerfectShutdown {
	ps := &PerfectShutdown{}
	ps.loop = 1

	go func() {
		signalchan := make(chan os.Signal)
		signal.Notify(signalchan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP)
		log.Println("accept stop command:", <-signalchan)
		if ps.before != nil {
			ps.before(ps.beforeparams)
		}
		atomic.StoreInt32(&ps.loop, 0)
	}()

	return ps
}

// IsClose 判断是否要关闭
func (ps *PerfectShutdown) IsClose() bool {
	return atomic.LoadInt32(&ps.loop) == 0
}

// Wait 判断是否要关闭
func (ps *PerfectShutdown) Wait(tm time.Duration) bool {
	now := time.Now()
	for time.Now().Sub(now) <= tm {
		if ps.IsClose() {
			return false
		}
		time.Sleep(time.Second)
	}
	return true
}

// SetBefore 判断是否要关闭
func (ps *PerfectShutdown) SetBefore(do func(params interface{}), params interface{}) {
	ps.before = do
	ps.beforeparams = params
}
