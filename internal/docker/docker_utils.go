package dockerutils

import (
	"log"
	"sync"

	"github.com/docker/docker/client"
)

var (
	dockerCli *client.Client
	once      sync.Once
)

func GetDockerCli() *client.Client {
	once.Do(func() {
		var err error
		dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			log.Fatalf("Error creating Docker client: %v", err)
		}
	})
	return dockerCli
}
