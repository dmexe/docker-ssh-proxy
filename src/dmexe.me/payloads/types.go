package payloads

// Payload holds queries
type Payload struct {
	ContainerID    string
	ContainerEnv   string
	ContainerLabel string
}

// Parser generic interface
type Parser interface {
	Parse(string) (Payload, error)
}
