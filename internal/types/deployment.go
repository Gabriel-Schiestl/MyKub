package types

import (
	"context"
	"fmt"
	"strings"

	dockerutils "github.com/Gabriel-Schiestl/reverse-proxy/internal/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type Deployment struct {
	Image        string      `json:"image"`
	Containers   []Container `json:"containers"`
	Memory       int         `json:"memory"`
	CPU          int         `json:"cpu"`
	CurrentIndex int32
	ExposedPort string `json:"exposed_port"`
}

func NewDeployment(image string, memory int, cpu int) *Deployment {
	cli := dockerutils.GetDockerCli()
    ctx := context.Background()

    imageInspect, err := cli.ImageInspect(ctx, image)
    if err != nil {
        panic(fmt.Errorf("erro ao inspecionar imagem: %w", err))
    }

    var containerPort string
    for k := range imageInspect.Config.ExposedPorts {
        containerPort = string(k)
        break
    }

	if containerPort == "" {
        fmt.Println("Aviso: Imagem não expõe nenhuma porta, usando porta padrão 80")
        containerPort = "80"
    }

	return &Deployment{
		Image:      image,
		Containers: []Container{},
		Memory:     memory,
		CPU:        cpu,
		ExposedPort: containerPort,
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
		ID:         uuid.NewString(),
		UsedCPU:    0,
		UsedMemory: 0,
		Port:       port,
	}

	d.Containers = append(d.Containers, newContainer)

	cli := dockerutils.GetDockerCli()
	ctx := context.Background()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
    if err != nil {
        fmt.Printf("Erro ao listar containers: %v\n", err)
    } else {
        for _, c := range containers {
            if strings.Contains(c.Image, d.Image) {
                containerJSON, err := cli.ContainerInspect(ctx, c.ID)
                if err != nil {
                    fmt.Printf("Error inspecting container %s: %v\n", c.ID, err)
                    continue
                }

                for _, portBinding := range containerJSON.HostConfig.PortBindings {
                    for _, binding := range portBinding {
                        if binding.HostPort == fmt.Sprintf("%d", port) {
                            fmt.Printf("Removing existing container %s with port %d\n", c.ID[:12], port)
                            
                            timeout := 10
                            err = cli.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout})
                            if err != nil {
                                fmt.Printf("Error stoping container %s: %v\n", c.ID[:12], err)
                            }
                            
                            err = cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
                            if err != nil {
                                fmt.Printf("Error removing container %s: %v\n", c.ID[:12], err)
                            }
                        }
                    }
                }
            }
        }
    }

    containerPort := d.ExposedPort
    if !strings.Contains(containerPort, "/") {
        containerPort = fmt.Sprintf("%s/tcp", containerPort)
    }

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