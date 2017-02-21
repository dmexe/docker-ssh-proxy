package apiserver

import (
	"context"
	"daemon/utils"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
)

// ManagerOptions keeps parameters for a new manager instance
type ManagerOptions struct {
	Providers []Provider
	Interval  time.Duration
}

// Manager keeps internal tasks of a manager instance
type Manager struct {
	providers []Provider
	interval  time.Duration
	log       *logrus.Entry
	tasks     []Task
	counter   uint64
	lock      sync.Mutex
	ctx       context.Context
}

// NewManager creates a new manager with given options
func NewManager(ctx context.Context, opts ManagerOptions) (*Manager, error) {
	manager := &Manager{
		providers: opts.Providers,
		interval:  opts.Interval,
		log:       utils.NewLogEntry("api.manager"),
		ctx:       ctx,
	}

	return manager, nil
}

// Tasks returns tasks tasks
func (m *Manager) Tasks() []Task {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.tasks
}

// Run pooling
func (m *Manager) Run(wg *sync.WaitGroup) error {

	if err := m.load(); err != nil {
		return err
	}

	m.log.Infof("Manager started")

	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {

			case <-m.ctx.Done():
				m.log.Debug("Context done")
				return

			case <-time.After(m.interval):
				if err := m.load(); err != nil {
					m.log.Errorf("Could not load tasks (%s)", err)
				}
			}
		}
	}()

	return nil
}

func (m *Manager) load() error {
	result := make([]Task, 0)

	for _, provider := range m.providers {
		tasks, err := provider.LoadTasks(m.ctx)
		if err != nil {
			return err
		}
		result = append(result, tasks...)
	}

	m.log.Debugf("Load %d tasks", len(result))

	m.lock.Lock()
	defer m.lock.Unlock()

	m.tasks = result
	m.counter++

	return nil
}

func (m *Manager) getCounter() uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.counter
}
