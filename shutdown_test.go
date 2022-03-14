package perfectshutdown

import (
	"log"
	"testing"
	"time"
)

func TestClose(t *testing.T) {
	ps := New()
	ps.Wait(time.Second * 4)
	ps.Close()
}

func TestLoop(t *testing.T) {
	ps := New()
	ps.Loop(func(ps *PerfectShutdown) {
		ps.Wait(time.Second * 2)
	})
}

func TestKill(t *testing.T) {
	ps := New()

	for i := 0; i < 5; i++ {
		go func() {
			ps.Loop(func(ps *PerfectShutdown) {
				log.Println("Loop")
				ps.Wait(time.Second * 10)
			})
		}()
	}

	ps.Loop(func(ps *PerfectShutdown) {
		log.Println("Loop")
		ps.Wait(time.Second * 10)
	})

}
