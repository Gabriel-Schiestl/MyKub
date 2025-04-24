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
var removedContainers map[string][]string = make(map[string][]string)

type ContainerStatus struct {
    Failed    map[string][]string
    Recovered map[string][]string 
}

func HealthChecker(deployments map[string]*types.Deployment, ch chan ContainerStatus) {
    for {
        containersToCheck := make(map[string][]string)
        
        for key, deployment := range deployments {
            for _, container := range deployment.Containers {
                containerURL := fmt.Sprintf("http://localhost:%d", container.Port)
                containersToCheck[key] = append(containersToCheck[key], containerURL)
            }
        }

        for path, containers := range removedContainers {
            containersToCheck[path] = append(containersToCheck[path], containers...)
        }

        fmt.Println("Containers to check:", containersToCheck)
        
        status := ContainerStatus{
            Failed:    make(map[string][]string),
            Recovered: make(map[string][]string),
        }
        
        stillFailing := make(map[string][]string)

        for path, containers := range containersToCheck {
            for _, containerURL := range containers {
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
                for _, removedURL := range removedContainers[path] {
                    if removedURL == containerURL {
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
                    
                    status.Failed[path] = append(status.Failed[path], containerURL)
                    stillFailing[path] = append(stillFailing[path], containerURL)
                } else {
                    resp.Body.Close()
                    
                    if wasRemoved {
                        fmt.Printf("Container recovered: %s\n", containerURL)
                        status.Recovered[path] = append(status.Recovered[path], containerURL)
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

func removeContainer(path string, container string) {
	removedContainers[path] = append(removedContainers[path], container)
}