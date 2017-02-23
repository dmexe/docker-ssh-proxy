package payloads

// Payload holds queries
type Payload struct {
	ContainerID    string `json:"containerId"`
	ContainerEnv   string `json:"containerEnv"`
	ContainerLabel string `json:"containerLabel"`
}

// Parser generic interface
type Parser interface {
	Parse(string) (Payload, error)
}
