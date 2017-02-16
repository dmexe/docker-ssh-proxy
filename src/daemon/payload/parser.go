package payload

type Request struct {
	ContainerId    string
	ContainerEnv   string
	ContainerLabel string
}

type Parser interface {
	Parse(string) (*Request, error)
}
