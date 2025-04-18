package health_checker

import (
	"net/http"
	"slices"
	"time"
)

var removedContainers []string

func HealthChecker(containers []string, ch chan []string) {
	for {
		containersToCheck := removedContainers
		containersToCheck = append(containersToCheck, containers...)
		removedContainers = []string{}

		for _, c := range containersToCheck {
			resp, err := http.Get(c + "/hello")
			if err != nil || resp.StatusCode != http.StatusOK {
				containersToCheck = removeContainer(containersToCheck, c)
			} 
		}

		ch <- containersToCheck

		time.Sleep(5 * time.Second)
	}
}

func removeContainer(containers []string, container string) []string {
	for i, c := range containers {
		if c == container {
			removedContainers = append(removedContainers, c)
			return slices.Delete(containers, i, i+1)
		}
	}
	return containers
}