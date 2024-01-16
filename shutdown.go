// 可以多个调用Loop. 关闭后在这个域全部都关闭
package perfectshutdown

import (
	"fmt"
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
	waitId string
	ticker *time.Ticker
	stop   chan bool
	once   sync.Once
}

func newWait(tm time.Duration) *wait {
	return &wait{ticker: time.NewTicker(tm), stop: make(chan bool)}
}

func (w *wait) Stop() {
	w.once.Do(func() {
		// log.Println("once")
		w.ticker.Stop()
		close(w.stop)
	})
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
			signal.Stop(signalchan)
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
			w.Stop()
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

func (ps *PerfectShutdown) StopAllWait() {
	ps.waitmap.Range(func(key, value any) bool {
		w := value.(*wait)
		w.Stop()
		return true
	})
}

func (ps *PerfectShutdown) StopWait(skey string) {
	ps.waitmap.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), skey+"-") {
			w := value.(*wait)
			w.Stop()
		}
		return true
	})
}

// WaitKey 和  Wait 区别就是 Wait的key默认是"", 等待时间. 会把这个等待的时间记录到该对应的key上. 类似time.Sleep 但是这个可以接受中断的信号. 如果isContinue == false 信号退出, 或者 call Close()
func (ps *PerfectShutdown) WaitKey(key string, tm time.Duration) (isContinue bool) {
	return ps.waitKey(key, tm)
}

// Wait 等待时间. 类似time.Sleep 但是这个可以接受中断的信号. 如果isContinue == false 信号退出, 或者 call Close()
func (ps *PerfectShutdown) Wait(tm time.Duration) (isContinue bool) {
	return ps.waitKey("", tm)
}

// Wait 等待时间. 类似time.Sleep 但是这个可以接受中断的信号. 如果isContinue == false 信号退出, 或者 call Close()
func (ps *PerfectShutdown) waitKey(key string, tm time.Duration) (isContinue bool) {
	isContinue = false
	if ps.IsClose() {
		return
	}

	w := newWait(tm)
	w.waitId = fmt.Sprintf("%s-%d", key, atomic.AddUint64(&ps.waitcount, 1))
	ps.waitmap.Store(w.waitId, w)
	defer func() {
		w.Stop()
		ps.waitmap.Delete(w.waitId)
	}()

	select {
	case <-w.ticker.C:
		isContinue = true
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
