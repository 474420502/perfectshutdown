package main

import (
	"log"
	"os"
	"time"

	"github.com/474420502/perfectshutdown"
)

func main() {
	ps := perfectshutdown.New()

	log.Println(os.Getpid())

	for i := 0; i < 5; i++ {
		go func() {
			ps.Loop(func(ps *perfectshutdown.PerfectShutdown) {
				log.Println("Loop")
				ps.Wait(time.Second * 10)
			})
		}()
	}

	ps.Loop(func(ps *perfectshutdown.PerfectShutdown) {
		log.Println("Loop")
		ps.Wait(time.Second * 10)
	})

}
