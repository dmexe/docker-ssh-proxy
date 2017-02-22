package payloads

// Payload holds queries
type Payload struct {
	ContainerID    string `json:"container_id"`
	ContainerEnv   string `json:"container_env"`
	ContainerLabel string `json:"container_label"`
}

// Parser generic interface
type Parser interface {
	Parse(string) (Payload, error)
}
