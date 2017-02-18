package main

import (
	"daemon/handlers"
	"daemon/payloads"
	"daemon/sshd"
	"flag"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

var (
	privateKeyFile string
	listenAddress  string
	debug          bool
)

func init() {
	flag.StringVar(&privateKeyFile, "sshd.pkey", "./id_rsa", "host private key file")
	flag.StringVar(&listenAddress, "sshd.listen", "0.0.0.0:2200", "listen address")
	flag.BoolVar(&debug, "debug", false, "enable debug output")
	flag.Parse()
}

func handleSignals() chan os.Signal {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	return signals
}

func main() {

	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug output enabled")
	}

	jwtParser, err := payloads.NewJwtParserFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	dockerClient, err := handlers.NewDockerClient()
	if err != nil {
		log.Fatal(err)
	}

	handler := func(exec string) (handlers.Handler, error) {
		payload, err := jwtParser.Parse(exec)
		if err != nil {
			return nil, err
		}
		return handlers.NewDockerHandler(dockerClient, payload)
	}

	serverOptions := sshd.CreateServerOptions{
		PrivateKeyFile: privateKeyFile,
		ListenAddr:     listenAddress,
	}

	server, err := sshd.NewServer(serverOptions, handler)
	if err != nil {
		log.Fatal(err)
	}

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}

	go func() {
		for sig := range handleSignals() {
			log.Infof("Got %s signal", sig)
			server.Close()
		}
	}()

	if err := server.Wait(); err != nil {
		log.Fatal(err)
	}
}
