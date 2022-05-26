package perfectshutdown

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type wait struct {
	count  uint64
	ticker *time.Ticker
	stop   chan bool
	once   sync.Once
}

func newWait(tm time.Duration) *wait {
	return &wait{ticker: time.NewTicker(tm), stop: make(chan bool)}
}

// PerfectShutdown 完美关闭程序
type PerfectShutdown struct {
	loop int32

	waitmap   sync.Map
	waitcount uint64

	beforeparams interface{}
	before       func(params interface{})
}

var ps = &PerfectShutdown{loop: 1}
var once sync.Once

// New 创建完美关闭程序
func New() *PerfectShutdown {

	once.Do(func() {
		go func() {
			signalchan := make(chan os.Signal, 1)
			signal.Notify(signalchan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP)
			log.Println("accept stop command:", <-signalchan, " --> wait to shutdown")
			if ps.before != nil {
				ps.before(ps.beforeparams)
			}
			atomic.StoreInt32(&ps.loop, 0)
			time.Sleep(time.Second)
			ps.waitmap.Range(func(key, value interface{}) bool {
				w := value.(*wait)
				w.once.Do(func() {
					// log.Println("once")
					w.ticker.Stop()
					close(w.stop)
				})
				return true
			})

		}()
	})

	return ps
}

func (ps *PerfectShutdown) Loop(do func(index int, ps *PerfectShutdown)) error {

	for n := 0; !ps.IsClose(); n++ {
		do(n, ps)
	}

	return nil
}

// IsClose 判断是否要关闭
func (ps *PerfectShutdown) IsClose() bool {
	return atomic.LoadInt32(&ps.loop) == 0
}

// Close 主动关闭
func (ps *PerfectShutdown) Close() {
	atomic.StoreInt32(&ps.loop, 0)

	log.Println("perfectshutdown: call Close() --> close")

	for i := 1; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		Func := runtime.FuncForPC(pc)
		if strings.HasPrefix(Func.Name(), "runtime.") || !ok {
			break
		}
		log.Printf("%s:%d func %s", file, line, Func.Name())
	}

}

// Wait 等待时间. 类似time.Sleep 但是这个可以接受中断的信号. 如果ok == false 信号退出, 或者 call Close()
func (ps *PerfectShutdown) Wait(tm time.Duration) (ok bool) {
	ok = false
	if ps.IsClose() {
		return
	}

	w := newWait(tm)
	w.count = atomic.AddUint64(&ps.waitcount, 1)
	ps.waitmap.Store(w.count, w)

	defer func() {
		w.once.Do(func() {
			w.ticker.Stop()
			close(w.stop)
			ps.waitmap.Delete(w.count)
		})
	}()

	select {
	case <-w.ticker.C:
		ok = true
		return
	case <-w.stop:
		return
	}

}
