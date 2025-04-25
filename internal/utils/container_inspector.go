package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	dockerutils "github.com/Gabriel-Schiestl/reverse-proxy/internal/docker"
	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func Inspect(deployments map[string]*types.Deployment, mu *sync.RWMutex) {
	mu.RLock()
	defer mu.RUnlock()

	cli := dockerutils.GetDockerCli()

	for _, deployment := range deployments {
		for _, container := range deployment.Containers {
			resp, err := cli.ContainerInspect(context.Background(), container.ID)
			if err != nil {
				log.Printf("Error inspecting container %s: %v", container.ID, err)
				continue
			}

			fmt.Println(resp.HostConfig.MemorySwap)
		}
	}
}

type ContainerStats struct {
    CPUPercentage    int
    MemoryPercentage int
}

func getContainerStats(cli *client.Client, ctx context.Context, containerType *types.Container, cpuLimit, memoryLimit int) (*ContainerStats, error) {
    stats, err := cli.ContainerStats(ctx, containerType.ID, false)
    if err != nil {
        return nil, err
    }
    defer stats.Body.Close()

    var statsJSON container.StatsResponse
    decoder := json.NewDecoder(stats.Body)
    if err := decoder.Decode(&statsJSON); err != nil {
        return nil, err
    }

    cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
            systemDelta := float64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
            
            cpuUnitsUsed := 0
            if systemDelta > 0 && cpuDelta > 0 {
                cpuUnitsUsed = int((cpuDelta / systemDelta) * float64(len(statsJSON.CPUStats.CPUUsage.PercpuUsage)) * 100.0)
            }
            
            cpuPercentage := 0
            if cpuLimit > 0 {
                cpuPercentage = (cpuUnitsUsed * 100) / cpuLimit
            }
            
            memoryUsedMB := int(statsJSON.MemoryStats.Usage / (1024 * 1024))
            
            memoryPercentage := 0
            if memoryLimit > 0 {
                memoryPercentage = (memoryUsedMB * 100) / memoryLimit
            }

            containerType.UsedCPU = cpuPercentage
            containerType.UsedMemory = memoryPercentage

    return &ContainerStats{
        CPUPercentage:    int(cpuPercentage),
        MemoryPercentage: int(memoryPercentage),
    }, nil
}