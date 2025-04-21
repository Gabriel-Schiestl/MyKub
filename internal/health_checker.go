package health_checker

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
)

//[path]: []containers
var removedContainers map[string][]string = make(map[string][]string)

func HealthChecker(deployments map[string]*types.Deployment, ch chan map[string][]string) {
	for {
		containersToCheck := removedContainers

		for key, deployment := range deployments {
			containers := []string{}
			for _, container := range deployment.Containers {
				containers = append(containers, "http://localhost:"+strconv.Itoa(container.Port))
			}
			containersToCheck[key] = append(containersToCheck[key], containers...)
		}

		removedContainers = make(map[string][]string)

		for path, containers := range containersToCheck {
			for _, container := range containers {
				resp, err := http.Get(container + "/hello")
				if err != nil || resp.StatusCode != http.StatusOK {
					removeContainer(path, container)
				}
			}
		}

		ch <- removedContainers

		time.Sleep(5 * time.Second)
	}
}

func removeContainer(path string, container string) {
	removedContainers[path] = append(removedContainers[path], container)
}