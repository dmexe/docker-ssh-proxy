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

	t.Run("should run and load tasks", func(t *testing.T) {
		var wg sync.WaitGroup

		state := []Task{{
			ID: "id",
		}}
		provider := &testProvider{
			state: state,
		}
		options := TasksManagerOptions{
			Providers: []Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(ctx)

		manager, err := NewManager(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, manager)

		require.NoError(t, manager.Run(&wg))
		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(2), manager.getCounter())
		require.Len(t, manager.GetTasks(), 1)

		cancel()
		wg.Wait()
	})

	t.Run("fail to load tasks", func(t *testing.T) {
		var wg sync.WaitGroup

		provider := &testProvider{
			err: errors.New("Boom"),
		}

		options := TasksManagerOptions{
			Providers: []Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(ctx)

		manager, err := NewManager(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, manager)
		require.EqualError(t, manager.Run(&wg), "Boom")
		require.Equal(t, uint64(0), manager.getCounter())
		require.Empty(t, manager.GetTasks())

		cancel()
		wg.Wait()
	})

	t.Run("should run but fail on background loading", func(t *testing.T) {
		var wg sync.WaitGroup

		state := []Task{{
			ID: "id",
		}}

		provider := &testProvider{
			state: state,
		}

		options := TasksManagerOptions{
			Providers: []Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(ctx)

		manager, err := NewManager(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, manager)

		require.NoError(t, manager.Run(&wg))

		provider.Lock()
		provider.err = errors.New("Boom")
		provider.Unlock()

		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(1), manager.getCounter())
		require.Len(t, manager.GetTasks(), 1)

		cancel()
		wg.Wait()
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
