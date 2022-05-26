# 优雅停机

* 接收到停机信号会等待循环结束. 然后退出
* IsClose() 返回 true. 自定义去处理. 带Wait函数去暂停

```go
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
			ps.Loop(func(i int, ps *perfectshutdown.PerfectShutdown) {
				log.Println(i, "Loop")
				ps.Wait(time.Second * 10)
			})
		}()
	}

	ps.Loop(func(i int, ps *perfectshutdown.PerfectShutdown) {
		log.Println(i, "Loop")
		ps.Wait(time.Second * 10)
	})

}

```


```bash
kill {pid}
accept stop command: terminated  --> wait to shutdown
```