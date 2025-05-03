package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	health_checker "github.com/Gabriel-Schiestl/reverse-proxy/internal"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/proxy"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/utils"
)

func main() {
	deployments := make(map[string]*types.Deployment) 
	var deploymentsMu sync.RWMutex
	var containerPort int = 8080

	containersStatus := make(chan health_checker.ContainerStatus)

	proxy := proxy.NewProxy(deployments, &deploymentsMu)

	go health_checker.HealthChecker(deployments, containersStatus)

	go utils.Update(&deploymentsMu, deployments, containersStatus, &containerPort)

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
