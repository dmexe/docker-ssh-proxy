package utils

import (
	"time"
	"github.com/Sirupsen/logrus"
	"context"
	"sync"
)

// BytesBroker keeps listeners and broadcast messages,
// https://gist.github.com/schmohlio/d7bdb255ba61d3f5e51a512a7c0d6a85#file-sse-go-L50
type BytesBroker struct {
	// Events are pushed to this channel by the main events-gathering routine
	notifier chan []byte

	// New client connections
	newClients chan chan []byte

	// Closed client connections
	closingClients chan chan []byte

	// Client connections registry
	clients map[chan []byte]bool

	log *logrus.Entry
	ctx context.Context
}

// NewBytesBroker creates a new broker instance using context
func NewBytesBroker(ctx context.Context) *BytesBroker {
	return &BytesBroker{
		notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte, 1),
		closingClients: make(chan chan []byte, 1),
		clients:        make(map[chan []byte]bool),
		log:            NewLogEntry("utils.broker"),
		ctx:            ctx,
	}
}

// Nofity broadcasts bytes to all clients
func (b *BytesBroker) Notify(bytes []byte) {
	b.notifier <- bytes
}

// Add a new client
func (b *BytesBroker) Add(client chan []byte) {
	b.newClients <- client
}

// Remove client
func (b *BytesBroker) Remove(client chan []byte) {
	b.closingClients <- client
}

func (b *BytesBroker) length() int {
	return len(b.clients)
}

// Run starts broker
func (b *BytesBroker) Run(wg *sync.WaitGroup) error {
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case <- b.ctx.Done():
				b.log.Debug("Context done")
				return

			case s := <-b.newClients:
				b.clients[s] = true
				b.log.Debugf("Client added. %d registered clients", b.length())

			case s := <-b.closingClients:
				delete(b.clients, s)
				b.log.Debugf("Client removed. %d registered clients", b.length())

			case event := <-b.notifier:
				for clientMessageChan, _ := range b.clients {
					select {
					case clientMessageChan <- event:
					case <-time.After(time.Second):
						b.log.Warn("Skipping client")
					}
				}
			}
		}
	}()

	b.log.Debugf("Broker started")

	return nil
}
