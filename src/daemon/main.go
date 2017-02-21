package main

import (
	"daemon/utils"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func goRunner(r utils.Runnable, complete chan error) {
	if err := r.Run(); err != nil {
		complete <- err
		return
	}

	if err := r.Wait(); err != nil {
		complete <- err
		return
	}

	complete <- nil
}

func main() {
	cfg := newAppConfig()
	cfg.parseArgs()

	if err := cfg.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	if cfg.debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug output enabled")
	}

	complete := make(chan error, 3)

	if cfg.api.enabled {
		apiManager := cfg.getAPIManager()
		go goRunner(apiManager, complete)
	}

	if cfg.shell.enabled {
		payloadParser := cfg.getPayloadParser()
		dockerClient := cfg.getDockerClient()
		dockerShellHandler := cfg.getDockerShellHandler(dockerClient)
		privateKey := cfg.getPrivateKey()
		shellServer := cfg.getShellServer(privateKey, dockerShellHandler, payloadParser)
		go goRunner(shellServer, complete)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signals:
		log.Infof("Got %s signal", sig)

	case err := <-complete:
		if err != nil {
			log.Error(err)
		}
	}
}
