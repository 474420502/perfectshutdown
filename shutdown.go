// 可以多个调用Loop. 关闭后在这个域全部都关闭
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
	loop          int32
	loopWaitGruop sync.WaitGroup

	waitmap   sync.Map
	waitcount uint64

	stopOnce sync.Once

	onBefore func()

	onClosed func()
}

// var ps *PerfectShutdown
var once sync.Once

// New 创建完美关闭程序
func New() *PerfectShutdown {

	ps := &PerfectShutdown{loop: 1}

	once.Do(func() {
		go func() {
			signalchan := make(chan os.Signal, 1)
			signal.Notify(signalchan, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
			log.Println("accept stop command:", <-signalchan, " --> wait to shutdown")
			ps.stopLoop()
		}()
	})

	return ps
}

func (ps *PerfectShutdown) Loop(do func(index uint64, ps *PerfectShutdown)) error {

	ps.loopWaitGruop.Add(1)
	defer func() {
		ps.loopWaitGruop.Done()
		ps.loopWaitGruop.Wait()
	}()

	for n := uint64(0); !ps.IsClose(); n++ {
		do(n, ps)
	}

	return nil
}

// IsClose 判断是否要关闭
func (ps *PerfectShutdown) IsClose() bool {
	return atomic.LoadInt32(&ps.loop) == 0
}

// IsClose 判断是否要关闭
func (ps *PerfectShutdown) stopLoop() {

	ps.stopOnce.Do(func() {
		if ps.onBefore != nil {
			ps.onBefore()
		}

		atomic.StoreInt32(&ps.loop, 0)
		ps.waitmap.Range(func(key, value interface{}) bool {
			w := value.(*wait)
			w.once.Do(func() {
				// log.Println("once")
				w.ticker.Stop()
				close(w.stop)
			})
			return true
		})

		go func() {
			ps.loopWaitGruop.Wait()
			if ps.onClosed != nil {
				ps.onClosed()
			}
		}()

	})

}

// Close 主动关闭
func (ps *PerfectShutdown) Close() {
	defer func() {
		ps.stopLoop()
	}()

	log.Println("perfectshutdown: call Close() --> close")

	for i := 1; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		Func := runtime.FuncForPC(pc)
		if strings.HasPrefix(Func.Name(), "runtime.") || !ok {
			break
		}
		log.Printf("showdown at: %s:%d func at: %s", file, line, Func.Name())
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

func (ps *PerfectShutdown) OnBeforeClose(do func()) {
	ps.onBefore = do
}

func (ps *PerfectShutdown) OnClosed(do func()) {
	ps.onClosed = do
}
