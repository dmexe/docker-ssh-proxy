package payload

type Filter struct {
	ContainerId    string
	ContainerEnv   string
	ContainerLabel string
}

type Parser interface {
	Parse(string) (*Filter, error)
}
