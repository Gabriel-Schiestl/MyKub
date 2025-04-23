package types

import (
	"context"
	"fmt"

	dockerutils "github.com/Gabriel-Schiestl/reverse-proxy/internal/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

type Deployment struct {
	Image        string      `json:"image"`
	Containers   []Container `json:"containers"`
	Memory       int         `json:"memory"`
	CPU          int         `json:"cpu"`
	CurrentIndex int32
}

func NewDeployment(image string, memory int, cpu int) *Deployment {
	return &Deployment{
		Image:      image,
		Containers: []Container{},
		Memory:     memory,
		CPU:        cpu,
	}
}

type Container struct {
	ID         string `json:"id"`
	UsedCPU    int    `json:"used_cpu"`
	UsedMemory int    `json:"used_memory"`
	Port       int    `json:"port"`
}

func (d *Deployment) AddContainer(port int) {
	newContainer := Container{
		ID:         "Teste",
		UsedCPU:    0,
		UsedMemory: 0,
		Port:       port,
	}

	d.Containers = append(d.Containers, newContainer)

	cli := dockerutils.GetDockerCli()
	ctx := context.Background()

	containerPort := fmt.Sprintf("%d/tcp", port)

	portBindings := nat.PortMap{
		nat.Port(containerPort): []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", port),
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: d.Image,
		ExposedPorts: nat.PortSet{
			nat.Port(containerPort): struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: portBindings,
		Resources: container.Resources{
			Memory:     int64(d.Memory * 1024 * 1024),
			CPUShares:  int64(d.CPU),
		},
	}, &network.NetworkingConfig{}, nil, "")
	if err != nil {
		panic(err)
	}

	newContainer.ID = resp.ID

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}
}