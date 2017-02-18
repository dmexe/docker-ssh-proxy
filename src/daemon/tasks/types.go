package tasks

import "net"

// Provider is an interface for task loaders
type Provider interface {
	LoadTasks() ([]Task, error)
}

// Task keeps exported task fields and instances
type Task struct {
	ID          string
	Limits      map[string]string
	Constraints map[string]string
	Version     string
	Instances   []Instance
}

// Instance keeps exported instance fields
type Instance struct {
	ID             string
	Addr           net.Addr
	State          string
	Healthy        bool
	ContainerID    string
	ContainerLabel string
	ContainerEnv   string
}
