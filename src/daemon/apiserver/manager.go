package apiserver

import (
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
	completed chan error
	stop      chan bool
	tasks     []Task
	counter   uint64
	running   bool
	lock      sync.Mutex
}

// NewManager creates a new manager with given options
func NewManager(opts ManagerOptions) (*Manager, error) {
	manager := &Manager{
		providers: opts.Providers,
		interval:  opts.Interval,
		completed: make(chan error),
		stop:      make(chan bool, 1),
		log:       utils.NewLogEntry("api.manager"),
	}

	return manager, nil
}

// Tasks returns tasks tasks
func (m *Manager) Tasks() []Task {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.tasks
}

// Wait for all jobs complete
func (m *Manager) Wait() error {
	if !m.isRunning() {
		m.log.Warnf("Could not wait, manager isn't running")
		return nil
	}

	err := <-m.completed
	if err == nil {
		m.log.Info("Manager completed")
	}
	return err
}

// Close manager, stop all jobs
func (m *Manager) Close() error {
	if !m.isRunning() {
		m.log.Warnf("Could not close, manager isn't running")
		return nil
	}

	m.stop <- true

	return m.Wait()
}

// Run pooling
func (m *Manager) Run() error {

	if err := m.load(); err != nil {
		return err
	}

	m.log.Infof("Manager sucessfully started")

	m.running = true

	go func() {
		for {
			select {

			case <-m.stop:
				m.completed <- nil
				return

			case <-time.After(m.interval):
				if err := m.load(); err != nil {
					m.completed <- err
					return
				}

			}
		}
	}()

	return nil
}

func (m *Manager) load() error {
	result := make([]Task, 0)

	for _, provider := range m.providers {
		tasks, err := provider.LoadTasks()
		if err != nil {
			return err
		}
		result = append(result, tasks...)
	}

	m.log.Debugf("Load %d tasks", len(result))

	m.lock.Lock()
	m.tasks = result
	m.counter++
	m.lock.Unlock()
	return nil
}

func (m *Manager) getCounter() uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.counter
}

func (m *Manager) isRunning() bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.running
}
