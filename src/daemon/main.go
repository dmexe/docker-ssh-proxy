package main

import (
	"daemon/agent"
	"daemon/payload"
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
	flag.StringVar(&privateKeyFile, "k", "./id_rsa", "host private key file")
	flag.StringVar(&listenAddress, "l", "0.0.0.0:2200", "listen address")
	flag.BoolVar(&debug, "d", false, "enable debug output")
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

	jwtParser, err := payload.NewJwtParserFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	dockerClient, err := agent.NewDockerClient()
	if err != nil {
		log.Fatal(err)
	}

	handler := func(payload string) (agent.Handler, error) {
		filter, err := jwtParser.Parse(payload)
		if err != nil {
			return nil, err
		}
		return agent.NewDockerHandler(dockerClient, filter)
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
