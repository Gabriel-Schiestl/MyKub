package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
)

func NewProxy(deployments map[string]*types.Deployment, mu *sync.RWMutex) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			url := r.URL.Path

			for path, deployment := range deployments {
				if strings.HasPrefix(url, path) {
					index := atomic.LoadInt32(&deployment.CurrentIndex)

					mu.RLock()
					container := deployment.Containers[index%int32(len(deployment.Containers))]
					r.URL.Scheme = "http"
					r.URL.Host = fmt.Sprintf("localhost:%d", container.Port)
					r.URL.Path = strings.TrimPrefix(url, path)

					if r.URL.Path == "" {
						r.URL.Path = "/"
					}
					atomic.AddInt32(&deployment.CurrentIndex, 1)
					mu.RUnlock()

					return
				}
			}
		},
	}
}
