package aggregator

import (
	"context"
	"daemon/utils"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
	"daemon/apiserver"
)

// TasksManagerOptions keeps parameters for a new manager instance
type ProviderOptions struct {
	Providers []apiserver.Provider
	Interval  time.Duration
}

// TasksManager keeps internal tasks of a manager instance
type Provider struct {
	providers []apiserver.Provider
	interval  time.Duration
	log       *logrus.Entry
	tasks     []apiserver.Task
	counter   uint64
	lock      sync.Mutex
	ctx       context.Context
}

// NewManager creates a new manager with given options
func NewProvider(ctx context.Context, opts ProviderOptions) (*Provider, error) {
	manager := &Provider{
		providers: opts.Providers,
		interval:  opts.Interval,
		log:       utils.NewLogEntry("api.aggregator"),
		ctx:       ctx,
	}

	return manager, nil
}

// GetTasks returns collected tasks
func (p *Provider) GetTasks(_ context.Context) ([]apiserver.Task, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.tasks, nil
}

func (p *Provider) setTasks(tasks []apiserver.Task) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.tasks = tasks
	p.counter++
}

// Run pooling
func (p *Provider) Run(wg *sync.WaitGroup) error {

	if err := p.load(); err != nil {
		return err
	}

	p.log.Infof("TasksManager started")

	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {

			case <-p.ctx.Done():
				p.log.Debug("Context done")
				return

			case <-time.After(p.interval):
				if err := p.load(); err != nil {
					p.log.Errorf("Could not load tasks (%s)", err)
				}
			}
		}
	}()

	return nil
}

func (p *Provider) load() error {
	result := make([]apiserver.Task, 0)

	for _, provider := range p.providers {
		tasks, err := provider.GetTasks(p.ctx)
		if err != nil {
			return err
		}
		result = append(result, tasks...)
	}

	p.setTasks(result)
	p.log.Debugf("Load %d tasks", len(result))

	return nil
}

func (p *Provider) getCounter() uint64 {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.counter
}

