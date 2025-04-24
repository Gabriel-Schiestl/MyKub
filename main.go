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

	deployments["/teste"] = types.NewDeployment("back", 256, 300)
	deployments["/teste"].AddContainer(containerPort)

	containersUnavailable := make(chan map[string][]string)

	proxy := proxy.NewProxy(deployments, &deploymentsMu)

	go health_checker.HealthChecker(deployments, containersUnavailable)

	go func() {
		for containerList := range containersUnavailable {
			deploymentsMu.Lock()
			for path, containers := range containerList {
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
			deploymentsMu.Unlock()

			log.Println("Updated containers:", deployments)
		}
	}()

	http.Handle("/", proxy)
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

			deploymentsMu.Lock()
			deployment.Deployment.AddContainer(containerPort)
			deployments[deployment.Path] = &deployment.Deployment
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
	

	log.Fatal(http.ListenAndServe(":80", nil))
}
