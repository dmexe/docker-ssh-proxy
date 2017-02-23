package utils

import (
	"testing"
	"context"
	"sync"
	"github.com/stretchr/testify/require"
	"github.com/Sirupsen/logrus"
)

func Test_BytesBroker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	payload := []byte{1}

	logrus.SetLevel(logrus.DebugLevel)

	broker := NewBytesBroker(ctx)
	require.NoError(t, broker.Run(&wg))

	t.Run("should register/deregister clients", func(t *testing.T) {
		var msg []byte

		// add first client
		clientA := make(chan []byte, 1)
		broker.Add(clientA)
		go broker.Notify(payload)

		// receive message
		msg = <- clientA
		require.Equal(t, payload, msg)
		require.Equal(t, 1, broker.length())

		// add second client
		clientB := make(chan []byte, 1)
		broker.Add(clientB)
		go broker.Notify(payload)

		// receive
		msg = <- clientA
		require.Equal(t, payload, msg)

		// receive
		msg = <- clientB
		require.Equal(t, payload, msg)

		// remove first client
		require.Equal(t, 2, broker.length())
		broker.Remove(clientA)

		go broker.Notify(payload)

		// receive
		msg = <- clientB
		require.Equal(t, payload, msg)
		require.Equal(t, 1, broker.length())

		broker.Remove(clientB)
	})

	cancel()
	wg.Done()
}
