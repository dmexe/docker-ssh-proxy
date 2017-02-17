package payloads

type Payload struct {
	ContainerID    string
	ContainerEnv   string
	ContainerLabel string
}

type Parser interface {
	Parse(string) (*Payload, error)
}
