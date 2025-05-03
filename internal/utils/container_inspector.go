package utils

import (
	"context"
	"encoding/json"

	"github.com/Gabriel-Schiestl/reverse-proxy/internal/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

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