package health_checker

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
)

//[path]: []containers


type ContainerStatus struct {
    Failed    map[string][]*types.Container
    Recovered map[string][]*types.Container 
}

func HealthChecker(deployments map[string]*types.Deployment, ch chan ContainerStatus) {
    var removedContainers map[string][]*types.Container = make(map[string][]*types.Container)
    
    for {
        containersToCheck := make(map[string][]*types.Container)
        
        for key, deployment := range deployments {
            containersToCheck[key] = append(containersToCheck[key], deployment.Containers...)
        }

        for path, containers := range removedContainers {
            containersToCheck[path] = append(containersToCheck[path], containers...)
        }

        fmt.Println("Containers to check:", containersToCheck)
        
        status := ContainerStatus{
            Failed:    make(map[string][]*types.Container),
            Recovered: make(map[string][]*types.Container),
        }
        
        stillFailing := make(map[string][]*types.Container)

        for path, containers := range containersToCheck {
            for _, container := range containers {
                containerURL := fmt.Sprintf("http://localhost:%d", container.Port)
                fmt.Println("Checking container:", containerURL)
                
                _, err := url.Parse(containerURL)
                if err != nil {
                    fmt.Printf("Invalid container URL %s: %v\n", containerURL, err)
                    continue
                }
                
                healthEndpoint := joinURLPath(containerURL, "/hello")
                
                client := http.Client{
                    Timeout: 5 * time.Second,
                }
                
                resp, err := client.Get(healthEndpoint)
                
                wasRemoved := false
                for _, removed := range removedContainers[path] {
                    if removed.ID == container.ID {
                        wasRemoved = true
                        break
                    }
                }
                
                if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
                    if err != nil {
                        fmt.Printf("Health check failed for %s: %v\n", healthEndpoint, err)
                    } else {
                        fmt.Printf("Health check returned non-OK status: %d for %s\n", resp.StatusCode, healthEndpoint)
                        resp.Body.Close()
                    }
                    
                    removeContainer(path, container, status, stillFailing)
                } else {
                    resp.Body.Close()
                    
                    if wasRemoved {
                        fmt.Printf("Container recovered: %s\n", containerURL)
                        status.Recovered[path] = append(status.Recovered[path], container)
                    }
                }
            }
        }
        
        removedContainers = stillFailing
        
        ch <- status
        
        time.Sleep(10 * time.Second)
    }
}

func joinURLPath(baseURL, path string) string {
    baseURL = strings.TrimSuffix(baseURL, "/")
    path = strings.TrimPrefix(path, "/")
    return baseURL + "/" + path
}

func removeContainer(path string, container *types.Container, status ContainerStatus, stillFailing map[string][]*types.Container) {
    fmt.Printf("Removing container %s from path %s\n", container.ID, path)
	status.Failed[path] = append(status.Failed[path], container)
    stillFailing[path] = append(stillFailing[path], container)
}