package utils

import (
	"context"
	"log"
	"slices"
	"sort"
	"sync"
	"time"

	health_checker "github.com/Gabriel-Schiestl/reverse-proxy/internal"
	dockerutils "github.com/Gabriel-Schiestl/reverse-proxy/internal/docker"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
	"github.com/docker/docker/api/types/container"
)

func Update(deploymentsMu *sync.RWMutex, deployments map[string]*types.Deployment, containersStatus chan health_checker.ContainerStatus, containerPort *int) {
	lastAutoScale := make(map[string]time.Time)
	
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

		now := time.Now()
        for path, deployment := range deployments {
            if lastAutoScaled, exists := lastAutoScale[path]; !exists || now.Sub(lastAutoScaled) > 2*time.Minute {
                checkAndAutoScale(deployment, path, deploymentsMu, containerPort)
                lastAutoScale[path] = now
            }
        }

		deploymentsMu.Unlock()

		log.Println("Updated containers:", deployments)
	}
}

func checkAndAutoScale(deployment *types.Deployment, path string, deploymentsMu *sync.RWMutex, containerPort *int) {
    const (
        cpuScaleUpThreshold  = 70
        cpuScaleDownThreshold = 30 
        maxContainers        = 5
        minContainers        = 1
    )

    cli := dockerutils.GetDockerCli()
    ctx := context.Background()

    totalCPU := 0
    activeContainers := 0

    for _, container := range deployment.Containers {
        stats, err := getContainerStats(cli, ctx, container, deployment.CPU, deployment.Memory)
        if err != nil {
            log.Printf("Error getting stats for container %s: %v", container.ID, err)
            continue
        }

		totalCPU += stats.CPUPercentage
        activeContainers++
    }

    if activeContainers == 0 {
        return
    }

    avgCPU := totalCPU / activeContainers
    log.Printf("Deployment %s: Avg CPU: %d%%, Active containers: %d", path, avgCPU, activeContainers)
    
    if avgCPU > cpuScaleUpThreshold && len(deployment.Containers) < maxContainers {
        log.Printf("Auto-scaling up deployment %s (CPU: %d%%)", path, avgCPU)
        
        newPort := *containerPort
        *containerPort++
        
        deployment.AddContainer(newPort)
        log.Printf("Added container for %s on port %d", path, newPort)
    }
	if avgCPU < cpuScaleDownThreshold && len(deployment.Containers) > minContainers {
        log.Printf("Auto-scaling down deployment %s (CPU: %d%%)", path, avgCPU)
        
        lastContainer := deployment.Containers[len(deployment.Containers)-1]
        
        timeout := 10
        if err := cli.ContainerStop(ctx, lastContainer.ID, container.StopOptions{Timeout: &timeout}); err != nil {
            log.Printf("Error stopping container %s: %v", lastContainer.ID, err)
        } else if err := cli.ContainerRemove(ctx, lastContainer.ID, container.RemoveOptions{Force: true}); err != nil {
            log.Printf("Error removing container %s: %v", lastContainer.ID, err)
        } else {
            deployment.Containers = deployment.Containers[:len(deployment.Containers)-1]
            log.Printf("Removed container %s from %s", lastContainer.ID, path)
        }
    }
}