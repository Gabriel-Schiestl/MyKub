package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	health_checker "github.com/Gabriel-Schiestl/reverse-proxy/internal"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/proxy"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
)

func main() {
	deployments := make(map[string]*types.Deployment) 
	var deploymentsMu sync.RWMutex
	var containerPort int = 8080

	// deployments["/teste"] = types.NewDeployment("back", 256, 300)
	// deployments["/teste"].AddContainer(containerPort)

	containersStatus := make(chan health_checker.ContainerStatus)

	proxy := proxy.NewProxy(deployments, &deploymentsMu)

	go health_checker.HealthChecker(deployments, containersStatus)

	go func() {
		for status := range containersStatus {
			deploymentsMu.Lock()

			for path, containers := range status.Failed {
				deployment := deployments[path]

				indexesToRemove := []int{}

				for index, deployContainer := range deployment.Containers {
					for _, container := range containers {
						parsedURL, err := url.Parse(container)
						if err != nil {
							log.Printf("Error parsing URL %s: %v", container, err)
							continue
						}

						host := parsedURL.Host
						portStr := ""
						if strings.Contains(host, ":") {
							portStr = strings.Split(host, ":")[1]
						}
						
						port, err := strconv.Atoi(portStr)
						if err != nil {
							log.Printf("Error converting port %s: %v", portStr, err)
							continue
						}

						if port == deployContainer.Port {
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
				
				for _, containerURL := range containers {
					parsedURL, err := url.Parse(containerURL)
					if err != nil {
						log.Printf("Error parsing URL %s: %v", containerURL, err)
						continue
					}
					
					host := parsedURL.Host
					portStr := ""
					if strings.Contains(host, ":") {
						portStr = strings.Split(host, ":")[1]
					}
					
					port, err := strconv.Atoi(portStr)
					if err != nil {
						log.Printf("Error converting port %s: %v", portStr, err)
						continue
					}
					
					containerExists := false
					for _, existingContainer := range deployment.Containers {
						if existingContainer.Port == port {
							containerExists = true
							break
						}
					}
					
					if !containerExists {
						log.Printf("Re-adding recovered container on port %d to %s", port, path)
						deployment.AddContainer(port)
					}
				}
        	}

			deploymentsMu.Unlock()

			log.Println("Updated containers:", deployments)
		}
		
	}()

	http.HandleFunc("/deployment", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var deployment struct{
				Path string `json:"path"`
				Deployment types.Deployment `json:"deployment"`
			}
			if err := json.NewDecoder(r.Body).Decode(&deployment); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			newDeployment := types.NewDeployment(deployment.Deployment.Image, deployment.Deployment.Memory, deployment.Deployment.CPU)

			deploymentsMu.Lock()
			newDeployment.AddContainer(containerPort)
			deployments[deployment.Path] = newDeployment
			containerPort++
			deploymentsMu.Unlock()

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(deployment)
		} else if r.Method == http.MethodGet {
			deploymentsMu.RLock()
			json.NewEncoder(w).Encode(deployments)
			deploymentsMu.RUnlock()
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.Handle("/", proxy)
	
	log.Fatal(http.ListenAndServe(":80", nil))
}
