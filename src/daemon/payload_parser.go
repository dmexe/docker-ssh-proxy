package main

type Payload struct {
	ContainerId    string
	ContainerEnv   string
	ContainerLabel string
}

type PayloadParser interface {
	Parse(string) (*Payload, error)
}
