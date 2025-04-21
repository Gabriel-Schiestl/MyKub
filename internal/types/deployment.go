package types

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
	container := Container{
		ID:         "Teste",
		UsedCPU:    0,
		UsedMemory: 0,
		Port:       port,
	}

	d.Containers = append(d.Containers, container)
}