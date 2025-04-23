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

func HealthChecker(deployments map[string]*types.Deployment, ch chan map[string][]string) {
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
        
        removedContainers = make(map[string][]string)

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
                if err != nil {
                    fmt.Printf("Health check failed for %s: %v\n", healthEndpoint, err)
                    removeContainer(path, containerURL)
                    continue
                }
                
                if resp.StatusCode != http.StatusOK {
                    fmt.Printf("Health check returned non-OK status: %d for %s\n", resp.StatusCode, healthEndpoint)
                    removeContainer(path, containerURL)
                }
                
                resp.Body.Close()
            }
        }

        ch <- removedContainers
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