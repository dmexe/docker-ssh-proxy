package aggregator

import (
	"context"
	"dmexe.me/apiserver"
	"errors"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func Test_Aggregator(t *testing.T) {
	ctx := context.Background()

	t.Run("should run and get tasks", func(t *testing.T) {
		var wg sync.WaitGroup

		state := apiserver.Result{
			Tasks: []apiserver.Task{{
				ID: "id",
			}},
			Digest: "digest",
		}

		provider := &testProvider{
			state: state,
		}
		ctx, cancel := context.WithCancel(ctx)

		options := ProviderOptions{
			Providers: []apiserver.Provider{provider},
			Interval:  100 * time.Millisecond,
			Broker:    getTestBroker(ctx),
		}

		agg, err := NewProvider(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, agg)

		require.NoError(t, agg.Run(&wg))
		time.Sleep(120 * time.Millisecond)

		result, err := agg.GetTasks(ctx)
		require.NoError(t, err)
		require.Len(t, result.Tasks, 1)

		cancel()
		wg.Wait()
	})

	t.Run("fail to get tasks", func(t *testing.T) {
		var wg sync.WaitGroup

		provider := &testProvider{
			err: errors.New("Boom"),
		}

		ctx, cancel := context.WithCancel(ctx)

		options := ProviderOptions{
			Providers: []apiserver.Provider{provider},
			Interval:  100 * time.Millisecond,
			Broker:    getTestBroker(ctx),
		}

		agg, err := NewProvider(ctx, options)
		require.NoError(t, err)
		require.NotNil(t, agg)
		require.EqualError(t, agg.Run(&wg), "Boom")

		result, err := agg.GetTasks(ctx)
		require.NoError(t, err)
		require.Empty(t, result.Tasks)

		cancel()
		wg.Wait()
	})
}

type testProvider struct {
	sync.Mutex
	state apiserver.Result
	err   error
}

type testContextKey string

func (p *testProvider) GetTasks(_ context.Context) (apiserver.Result, error) {
	p.Lock()
	defer p.Unlock()
	if p.err != nil {
		return p.state, p.err
	}
	return p.state, nil
}

func getTestBroker(ctx context.Context) *apiserver.Broker {
	ctx = context.WithValue(ctx, testContextKey("name"), "test")
	return apiserver.NewBroker(ctx)
}
