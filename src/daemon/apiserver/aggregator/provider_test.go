package aggregator

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
	"daemon/apiserver"
)

func Test_Manager(t *testing.T) {
	ctx := context.Background()

	t.Run("should run and get tasks", func(t *testing.T) {
		var wg sync.WaitGroup

		state := []apiserver.Task{{
			ID: "id",
		}}
		provider := &testProvider{
			state: state,
		}
		options := ProviderOptions{
			Providers: []apiserver.Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(ctx)

		agg, err := NewProvider(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, agg)

		require.NoError(t, agg.Run(&wg))
		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(2), agg.getCounter())

		tasks, err := agg.GetTasks(ctx)
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		cancel()
		wg.Wait()
	})

	t.Run("fail to get tasks", func(t *testing.T) {
		var wg sync.WaitGroup

		provider := &testProvider{
			err: errors.New("Boom"),
		}

		options := ProviderOptions{
			Providers: []apiserver.Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(ctx)

		agg, err := NewProvider(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, agg)
		require.EqualError(t, agg.Run(&wg), "Boom")
		require.Equal(t, uint64(0), agg.getCounter())

		tasks, err := agg.GetTasks(ctx)
		require.NoError(t, err)
		require.Empty(t, tasks)

		cancel()
		wg.Wait()
	})

	t.Run("should run but fail on background loading", func(t *testing.T) {
		var wg sync.WaitGroup

		state := []apiserver.Task{{
			ID: "id",
		}}

		provider := &testProvider{
			state: state,
		}

		options := ProviderOptions{
			Providers: []apiserver.Provider{provider},
			Interval:  100 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(ctx)

		agg, err := NewProvider(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, agg)

		require.NoError(t, agg.Run(&wg))

		provider.Lock()
		provider.err = errors.New("Boom")
		provider.Unlock()

		time.Sleep(120 * time.Millisecond)

		require.Equal(t, uint64(1), agg.getCounter())

		tasks, err := agg.GetTasks(ctx)
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		cancel()
		wg.Wait()
	})
}

type testProvider struct {
	sync.Mutex
	state []apiserver.Task
	err   error
}

func (p *testProvider) GetTasks(_ context.Context) ([]apiserver.Task, error) {
	p.Lock()
	defer p.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	return p.state, nil
}

