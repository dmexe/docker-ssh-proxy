package main

import (
	"context"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	cfg := newAppConfig(ctx)
	cfg.parseArgs()

	if err := cfg.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	if cfg.debug.enabled {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug output enabled")
	}

	if cfg.api.enabled {
		broker := cfg.getBroker()
		if err := broker.Run(&wg); err != nil {
			log.Fatal(err)
		}

		provider := cfg.getAPIProvider(broker)
		if err := provider.Run(&wg); err != nil {
			log.Fatal(err)
		}

		server := cfg.getAPIServer(provider, broker)
		if err := server.Run(&wg); err != nil {
			log.Fatal(err)
		}
	}

	if cfg.shell.enabled {
		payloadParser := cfg.getPayloadParser()
		dockerClient := cfg.getDockerClient()
		dockerShellHandler := cfg.getDockerShellHandler(dockerClient)
		privateKey := cfg.getPrivateKey()
		shellServer := cfg.getShellServer(privateKey, dockerShellHandler, payloadParser)

		if err := shellServer.Run(&wg); err != nil {
			log.Fatal(err)
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signals
	log.Infof("Got %s signal", sig)

	cancel()
	wg.Wait()
}
