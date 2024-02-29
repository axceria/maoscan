/*
	(c) Yariya
*/

package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

var (
	imported      uint64
	checked       uint64
	success       uint64
	statusCodeErr uint64

	proxyErr   uint64
	timeoutErr uint64
)

func Stater() {
	for range time.Tick(time.Second) {
		fmt.Printf("[+] scanner: [\u001B[34m%d\u001B[39m] checker: [\u001B[34m%d\u001B[39m] Results: (Valid: \033[32m%d\033[39m, Invalid: \u001B[31m%d\u001B[39m, Error: \u001B[31m%d\u001B[39m, Timeout: \u001B[31m%d\u001B[39m) | Threads: \u001B[34m%d\u001B[39m\n",
			atomic.LoadUint64(&imported),
			atomic.LoadUint64(&checked),
			atomic.LoadUint64(&success),
			atomic.LoadUint64(&statusCodeErr),
			atomic.LoadUint64(&proxyErr),
			atomic.LoadUint64(&timeoutErr),
			atomic.LoadInt64(&Proxies.openHttpThreads),
		)
	}
}
