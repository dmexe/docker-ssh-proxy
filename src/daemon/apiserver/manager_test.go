package apiserver

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func Test_Manager(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup

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

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		manager, err := NewManager(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, manager)

		require.NoError(t, manager.Run(&wg))
		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(2), manager.getCounter())
		require.Len(t, manager.Tasks(), 1)
	})

	t.Run("fail to load tasks", func(t *testing.T) {
		provider := &testProvider{
			err: errors.New("Boom"),
		}

		options := ManagerOptions{
			Providers: []Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		manager, err := NewManager(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, manager)
		require.EqualError(t, manager.Run(&wg), "Boom")
		require.Equal(t, uint64(0), manager.getCounter())
		require.Empty(t, manager.Tasks())
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

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		manager, err := NewManager(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, manager)

		require.NoError(t, manager.Run(&wg))

		provider.Lock()
		provider.err = errors.New("Boom")
		provider.Unlock()

		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(1), manager.getCounter())
		require.Len(t, manager.Tasks(), 1)
	})
}

type testProvider struct {
	sync.Mutex
	state []Task
	err   error
}

func (p *testProvider) LoadTasks(_ context.Context) ([]Task, error) {
	p.Lock()
	defer p.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	return p.state, nil
}
