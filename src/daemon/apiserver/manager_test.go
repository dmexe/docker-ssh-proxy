package apiserver

import (
	"errors"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func Test_Manager(t *testing.T) {
	t.Run("should run and load tasks", func(t *testing.T) {
		state := []Task{{
			ID: "id",
		}}
		provider := &testProvider{
			state: state,
		}
		options := ManagerOptions{
			Providers: []Provider{provider},
			Interval:  100 * time.Millisecond,
		}
		manager, err := NewManager(options)
		require.NoError(t, err)
		require.NotNil(t, manager)

		require.NoError(t, manager.Run())
		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(2), manager.getCounter())
		require.Len(t, manager.Tasks(), 1)
		require.NoError(t, manager.Close())
	})

	t.Run("fail to load tasks", func(t *testing.T) {
		provider := &testProvider{
			err: errors.New("Boom"),
		}

		options := ManagerOptions{
			Providers: []Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		manager, err := NewManager(options)
		require.NoError(t, err)
		require.NotNil(t, manager)
		require.EqualError(t, manager.Run(), "Boom")
		require.Equal(t, uint64(0), manager.getCounter())
		require.Empty(t, manager.Tasks())
		require.NoError(t, manager.Close())
	})

	t.Run("should run but fail on background loading", func(t *testing.T) {
		state := []Task{{
			ID: "id",
		}}
		provider := &testProvider{
			state: state,
		}
		options := ManagerOptions{
			Providers: []Provider{provider},
			Interval:  100 * time.Millisecond,
		}
		manager, err := NewManager(options)
		require.NoError(t, err)
		require.NotNil(t, manager)

		require.NoError(t, manager.Run())

		provider.Lock()
		provider.err = errors.New("Boom")
		provider.Unlock()

		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(1), manager.getCounter())
		require.Len(t, manager.Tasks(), 1)
		require.EqualError(t, manager.Close(), "Boom")
	})
}

type testProvider struct {
	sync.Mutex
	state []Task
	err   error
}

func (p *testProvider) LoadTasks() ([]Task, error) {
	p.Lock()
	defer p.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	return p.state, nil
}
