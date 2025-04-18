package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
	"sync/atomic"

	health_checker "github.com/Gabriel-Schiestl/reverse-proxy/internal"
)

var containers []string
var containerIndex int32
var containerMU sync.RWMutex

func main() {
	newContainers := make(chan []string)

	go health_checker.HealthChecker(containers, newContainers)

	go func() {
		for containerList := range newContainers {
			containerMU.Lock()
			containers = containerList
			containerMU.Unlock()

			log.Println("Updated containers:", containers)
		}
	}()

	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			index := atomic.LoadInt32(&containerIndex)

			containerMU.RLock()
			r.URL.Scheme = "http"
			r.URL.Host = containers[index%int32(len(containers))]
			containerMU.RUnlock()

			atomic.AddInt32(&containerIndex, 1)
		},
	}

	http.Handle("/", proxy)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
