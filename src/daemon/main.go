package main

import (
	"daemon/apiserver"
	"daemon/apiserver/marathon"
	"daemon/payloads"
	"daemon/sshd"
	"daemon/sshd/handlers"
	"flag"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	privateKeyFile string
	listenAddress  string
	marathonURL    string
	debug          bool
)

func init() {
	flag.StringVar(&privateKeyFile, "sshd.pkey", "./id_rsa", "host private key file")
	flag.StringVar(&listenAddress, "sshd.listen", "0.0.0.0:2200", "listen address")
	flag.StringVar(&marathonURL, "marathon.url", "http://marathon.mesos:8080", "marathon api url")
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

	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	serverOptions := sshd.ServerOptions{
		PrivateKey:  privateKey,
		ListenAddr:  listenAddress,
		HandlerFunc: handler,
	}

	server, err := sshd.NewServer(serverOptions)
	if err != nil {
		log.Fatal(err)
	}

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}

	go func() {
		for sig := range handleSignals() {
			log.Infof("Got %s signal", sig)
			server.Close()
		}
	}()

	providerOptions := marathon.ProviderOptions{
		Endpoint: marathonURL,
	}
	provider, err := marathon.NewProvider(providerOptions)
	if err != nil {
		log.Fatal(err)
	}

	managerOptions := apiserver.ManagerOptions{
		Provider: provider,
		Timeout:  10 * time.Second,
	}

	manager, err := apiserver.NewManager(managerOptions)
	if err != nil {
		log.Fatal(err)
	}

	if err := manager.Run(); err != nil {
		log.Fatal(err)
	}

	if err := manager.Wait(); err != nil {
		log.Fatal(err)
	}
}
