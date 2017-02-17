package payloads

type Payload struct {
	ContainerId    string
	ContainerEnv   string
	ContainerLabel string
}

type Parser interface {
	Parse(string) (*Payload, error)
}
