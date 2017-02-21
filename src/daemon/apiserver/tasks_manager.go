package apiserver

import (
	"context"
	"daemon/utils"
	"github.com/Sirupsen/logrus"
	"sync"
	"time"
)

// TasksManagerOptions keeps parameters for a new manager instance
type TasksManagerOptions struct {
	Providers []Provider
	Interval  time.Duration
}

// TasksManager keeps internal tasks of a manager instance
type TasksManager struct {
	providers []Provider
	interval  time.Duration
	log       *logrus.Entry
	tasks     []Task
	counter   uint64
	lock      sync.Mutex
	ctx       context.Context
}

// NewManager creates a new manager with given options
func NewManager(ctx context.Context, opts TasksManagerOptions) (*TasksManager, error) {
	manager := &TasksManager{
		providers: opts.Providers,
		interval:  opts.Interval,
		log:       utils.NewLogEntry("api.manager"),
		ctx:       ctx,
	}

	return manager, nil
}

// GetTasks returns tasks tasks
func (m *TasksManager) GetTasks() []Task {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.tasks
}

func (m *TasksManager) setTasks(tasks []Task) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.tasks = tasks
	m.counter++
}

// Run pooling
func (m *TasksManager) Run(wg *sync.WaitGroup) error {

	if err := m.load(); err != nil {
		return err
	}

	m.log.Infof("TasksManager started")

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

func (m *TasksManager) load() error {
	result := make([]Task, 0)

	for _, provider := range m.providers {
		tasks, err := provider.LoadTasks(m.ctx)
		if err != nil {
			return err
		}
		result = append(result, tasks...)
	}

	m.setTasks(result)
	m.log.Debugf("Load %d tasks", len(result))

	return nil
}

func (m *TasksManager) getCounter() uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.counter
}
