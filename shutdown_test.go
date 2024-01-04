package perfectshutdown

import (
	"log"
	"sync"
	"testing"
	"time"
)

func TestClose(t *testing.T) {
	ps := New()
	go func() {
		time.Sleep(time.Second * 2)
		ps.Close()
	}()
	ok := ps.Wait(time.Second * 4)
	if ok {
		t.Errorf("Wait should return false after Close is called")
	}
	// 确认IsClose返回true
	if !ps.IsClose() {
		t.Errorf("IsClose should return true after Close is called")
	}
}

func TestCallbacks(t *testing.T) {
	beforeCalled := false
	onCloseCalled := false

	ps := New()
	ps.OnBeforeClose(func() {
		beforeCalled = true
	})
	ps.OnClosed(func() {
		onCloseCalled = true
	})

	go func() {
		time.Sleep(time.Second * 1)
		// 触发关闭流程
		ps.Close()
	}()

	ps.Loop(func(index uint64, ps *PerfectShutdown) {

	})

	time.Sleep(time.Millisecond * 100)
	// 验证回调函数是否被调用
	if !beforeCalled {
		t.Error("onBefore callback was not called during shutdown")
	}
	if !onCloseCalled {
		t.Error("onClosed callback was not called after shutdown")
	}
}

func TestPerfectShutdown(t *testing.T) {
	t.Run("Close", func(t *testing.T) {
		ps := New()
		closeCalled := false
		ps.OnClosed(func() {
			closeCalled = true
		})
		ps.Close()
		time.Sleep(time.Millisecond * 100) // 因为WaitGroup确实会慢一点
		log.Println(closeCalled)
		if !closeCalled {
			t.Error("OnClosed callback was not called after Close")
		}
	})

	t.Run("WaitAfterClose", func(t *testing.T) {
		ps := New()
		ps.Close()
		ok := ps.Wait(time.Second)
		if ok {
			t.Error("Wait should return false immediately after Close is called")
		}
	})

	t.Run("LoopExitWhenClose", func(t *testing.T) {
		var loopCount uint64
		ps := New()
		go func() {
			time.Sleep(100 * time.Millisecond)
			ps.Close()
		}()
		err := ps.Loop(func(index uint64, ps *PerfectShutdown) {
			loopCount++
			time.Sleep(50 * time.Millisecond)
			if loopCount > 5 {
				t.Error("Loop did not stop when IsClose became true")
			}
		})
		if err != nil {
			t.Errorf("Unexpected error from Loop: %v", err)
		}
	})

	t.Run("InterruptedWaitInLoop", func(t *testing.T) {
		ps := New()
		waitCalled := false
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			time.Sleep(100 * time.Millisecond)
			ps.Close()
		}()
		err := ps.Loop(func(index uint64, ps *PerfectShutdown) {
			defer wg.Done()
			waitCalled = true
			ps.Wait(time.Second)
		})

		wg.Wait()
		if !waitCalled {
			t.Error("Wait was not called inside Loop")
		}
		if err != nil {
			t.Errorf("Unexpected error from Loop: %v", err)
		}
	})

	t.Run("MultipleLoopsClose", func(t *testing.T) {
		const numLoops = 3
		wg := sync.WaitGroup{}
		wg.Add(numLoops)
		ps := New()
		go func() {
			time.Sleep(100 * time.Millisecond)
			ps.Close()
		}()

		for i := 0; i < numLoops; i++ {
			go func() {
				defer wg.Done()
				ps.Loop(func(index uint64, ps *PerfectShutdown) {})
			}()
		}

		wg.Wait()
	})

	t.Run("OnBeforeCallback", func(t *testing.T) {
		onBeforeCalled := false
		ps := New()
		ps.OnBeforeClose(func() {
			onBeforeCalled = true
		})

		ps.stopLoop()

		if !onBeforeCalled {
			t.Error("onBefore callback was not called during shutdown")
		}
	})
}

func TestPerfectShutdown_Loop(t *testing.T) {
	ps := New()

	counter := 0
	err := ps.Loop(func(index uint64, ps *PerfectShutdown) {
		counter++
		if counter == 3 {
			ps.Close()
		}
	})

	if err != nil {
		t.Errorf("Loop returned an error: %v", err)
	}

	if counter != 3 {
		t.Errorf("Expected counter to be 3, but got %d", counter)
	}
}

func TestPerfectShutdown_Wait(t *testing.T) {
	ps := New()

	go func() {
		time.Sleep(time.Second)
		ps.Close()
	}()

	start := time.Now()
	ok := ps.Wait(2 * time.Second)
	elapsed := time.Since(start)

	if ok {
		t.Errorf("Expected Wait to return true, but got false")
	}

	if elapsed < time.Second || elapsed > 2*time.Second {
		t.Errorf("Expected Wait to wait for approximately 2 seconds, but got %s", elapsed)
	}
}

func TestPerfectShutdown_Close(t *testing.T) {
	ps := New()

	go func() {
		time.Sleep(time.Second)
		ps.Close()
	}()

	start := time.Now()
	ps.Close()
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Errorf("Expected Close to wait for approximately 1 second, but got %s", elapsed)
	}
}

func TestPerfectShutdown_Wait2(t *testing.T) {
	ps := New()

	go func() {
		time.Sleep(time.Second)
		ps.Close()
	}()

	start := time.Now()

	ps.Loop(func(index uint64, ps *PerfectShutdown) {
		for i := 0; ; i++ {
			if !ps.Wait(time.Millisecond * 300) {
				break
			}
		}
	})

	elapsed := time.Since(start)

	if elapsed < time.Second || elapsed > 2*time.Second {
		t.Errorf("Expected Wait to wait for approximately 2 seconds, but got %s", elapsed)
	}
}
