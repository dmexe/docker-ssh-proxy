package aggregator

import (
	"context"
	"dmexe.me/apiserver"
	"dmexe.me/utils"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
)

// ProviderOptions keeps parameters for a new manager instance
type ProviderOptions struct {
	Providers []apiserver.Provider
	Interval  time.Duration
	Broker    *apiserver.Broker
}

// Provider keeps internal tasks of a manager instance
type Provider struct {
	providers []apiserver.Provider
	interval  time.Duration
	log       *logrus.Entry
	result    apiserver.Result
	ctx       context.Context
	broker    *apiserver.Broker
}

// NewProvider creates a new manager with given options
func NewProvider(ctx context.Context, opts ProviderOptions) (*Provider, error) {
	manager := &Provider{
		providers: opts.Providers,
		interval:  opts.Interval,
		log:       utils.NewLogEntry("api.aggregator"),
		ctx:       ctx,
		result: apiserver.Result{
			CreatedAt: time.Now(),
		},
		broker: opts.Broker,
	}

	return manager, nil
}

// GetTasks returns collected tasks
func (p *Provider) GetTasks(_ context.Context) (apiserver.Result, error) {
	return p.result, nil
}

// Run pooling
func (p *Provider) Run(wg *sync.WaitGroup) error {

	if err := p.load(); err != nil {
		return err
	}

	p.log.Infof("Aggregator started")

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
	collected := make([]apiserver.Task, 0)
	sums := make([]string, 0)

	for _, provider := range p.providers {
		tasks, err := provider.GetTasks(p.ctx)
		if err != nil {
			return err
		}
		collected = append(collected, tasks.Tasks...)
		sums = append(sums, tasks.Digest)
	}

	newDigest := utils.StringDigest(sums...)
	if newDigest != p.result.Digest {
		newResult := apiserver.Result{
			Tasks:     collected,
			Digest:    newDigest,
			CreatedAt: time.Now(),
		}

		p.result = newResult

		p.broker.Notify(apiserver.BrokerEvent{
			Name: "tasks.changed",
		})
		p.log.Debugf("Load %d tasks", len(collected))
	} else {
		p.log.Debug("Nothing changed")
	}

	return nil
}
