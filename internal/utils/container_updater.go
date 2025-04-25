package utils

import (
	"log"
	"slices"
	"sort"
	"sync"

	health_checker "github.com/Gabriel-Schiestl/reverse-proxy/internal"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
)

func Update(deploymentsMu *sync.RWMutex, deployments map[string]*types.Deployment, containersStatus chan health_checker.ContainerStatus) {
	for status := range containersStatus {
		deploymentsMu.Lock()

		for path, containers := range status.Failed {
			deployment := deployments[path]

			indexesToRemove := []int{}

			for index, deployContainer := range deployment.Containers {
				for _, container := range containers {

					if container.Port == deployContainer.Port {
						indexesToRemove = append(indexesToRemove, index)
					}
				}
			}

			sort.Ints(indexesToRemove)
			for i := len(indexesToRemove) - 1; i >= 0; i-- {
				indexToRemove := indexesToRemove[i]
				log.Println("Removing container:", deployment.Containers[indexToRemove].ID)
				deployment.Containers = slices.Delete(deployment.Containers, indexToRemove, indexToRemove+1)
			}
		}

		for path, containers := range status.Recovered {
			deployment := deployments[path]
			if deployment == nil {
				log.Printf("Deployment for path %s not found", path)
				continue
			}

			for _, container := range containers {
				containerExists := false
				for _, existingContainer := range deployment.Containers {
					if existingContainer.ID == container.ID {
						containerExists = true
						break
					}
				}

				if !containerExists {
					log.Printf("Re-adding recovered container of id %s to %s", container.ID, path)
					deployment.Containers = append(deployment.Containers, container)
				}
			}
		}

		deploymentsMu.Unlock()

		log.Println("Updated containers:", deployments)
	}
}