package apiserver

import (
	"daemon/utils"
	"github.com/Sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

// ManagerOptions keeps parameters for a new manager instance
type ManagerOptions struct {
	Provider Provider
	Timeout  time.Duration
}

// Manager keeps internal tasks of a manager instance
type Manager struct {
	sync.RWMutex
	provider  Provider
	timeout   time.Duration
	log       *logrus.Entry
	completed chan error
	stop      chan bool
	tasks     []Task
	counter   uint64
	running   bool
}

// NewManager creates a new manager with given options
func NewManager(opts ManagerOptions) (*Manager, error) {
	manager := &Manager{
		provider:  opts.Provider,
		timeout:   opts.Timeout,
		completed: make(chan error, 1),
		stop:      make(chan bool, 1),
		log:       utils.NewLogEntry("tasks.manager"),
	}

	return manager, nil
}

// Tasks returns tasks tasks
func (m *Manager) Tasks() []Task {
	m.RLock()
	defer m.RUnlock()

	return m.tasks
}

// Wait for all jobs complete
func (m *Manager) Wait() error {
	if !m.running {
		m.log.Warnf("Could not wait, manager isn't running")
		return nil
	}

	select {
	case err := <-m.completed:
		return err
	}
}

// Close manager, stop all jobs
func (m *Manager) Close() error {
	if !m.running {
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

	m.running = true

	go func() {
		for {
			select {
			case <-m.stop:
				m.completed <- nil
				return
			case <-time.After(m.timeout):
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
	tasks, err := m.provider.LoadTasks()
	if err != nil {
		return err
	}
	m.Lock()
	m.log.Debugf("Load %d tasks", len(tasks))
	m.tasks = tasks
	m.Unlock()
	atomic.AddUint64(&m.counter, 1)
	return nil
}

func (m *Manager) getCounter() uint64 {
	return atomic.LoadUint64(&m.counter)
}
