package marathon

import (
	"daemon/utils"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"time"
)

// Provider loads tasks from marathon
type Provider struct {
	endpoint  string
	log       *log.Entry
	transport *http.Transport
}

// ProviderOptions keeps options for constructor
type ProviderOptions struct {
	Endpoint string
}

// NewProvider creates a new marathon driver instance using given options
func NewProvider(options ProviderOptions) (*Provider, error) {
	m := &Provider{
		endpoint: options.Endpoint,
		log:      utils.NewLogEntry("marathon"),
		transport: &http.Transport{
			IdleConnTimeout: 60 * time.Second,
		},
	}
	return m, nil
}

// LoadTasks from marathon
func (d *Provider) LoadTasks() {
}
