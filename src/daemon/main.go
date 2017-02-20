package main

import (
	"daemon/apiserver"
	"daemon/apiserver/marathon"
	"daemon/payloads"
	"daemon/sshd"
	"daemon/sshd/handlers"
	"flag"
	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
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

	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug output enabled")
	}
}

func handleSignals() chan os.Signal {
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	return signals
}

func getPayloadParser() payloads.Parser {
	jwtParser, err := payloads.NewJwtParserFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return jwtParser
}

func getDockerClient() *docker.Client {
	dockerClient, err := handlers.NewDockerClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return dockerClient
}

func getDockerShellHandler(dockerClient *docker.Client, parser payloads.Parser) handlers.HandlerFunc {
	handler := func(exec string) (handlers.Handler, error) {
		payload, err := parser.Parse(exec)
		if err != nil {
			return nil, err
		}
		return handlers.NewDockerHandler(handlers.DockerHandlerOptions{
			Client:  dockerClient,
			Payload: payload,
		})
	}

	return handler
}

func getPrivateKey() []byte {
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		log.Fatal(err)
	}
	return privateKey
}

func getShellServer(privateKey []byte, handlerFunc handlers.HandlerFunc) *sshd.Server {
	serverOptions := sshd.ServerOptions{
		PrivateKey:  privateKey,
		ListenAddr:  listenAddress,
		HandlerFunc: handlerFunc,
	}

	server, err := sshd.NewServer(serverOptions)
	if err != nil {
		log.Fatal(err)
	}

	return server
}

func getAPIServerProvider() apiserver.Provider {
	providerOptions := marathon.ProviderOptions{
		Endpoint: marathonURL,
	}
	provider, err := marathon.NewProvider(providerOptions)
	if err != nil {
		log.Fatal(err)
	}
	return provider
}

func getAPIServerManager(provider apiserver.Provider) *apiserver.Manager {
	managerOptions := apiserver.ManagerOptions{
		Provider: provider,
		Timeout:  10 * time.Second,
	}

	manager, err := apiserver.NewManager(managerOptions)
	if err != nil {
		log.Fatal(err)
	}
	return manager
}

func main() {

	payloadParser := getPayloadParser()
	dockerClient := getDockerClient()
	dockerShellHandler := getDockerShellHandler(dockerClient, payloadParser)
	privateKey := getPrivateKey()
	shellServer := getShellServer(privateKey, dockerShellHandler)

	apiServerProvider := getAPIServerProvider()
	apiServerManager := getAPIServerManager(apiServerProvider)

	if err := shellServer.Run(); err != nil {
		log.Fatal(err)
	}

	go func() {
		for sig := range handleSignals() {
			log.Infof("Got %s signal", sig)
			shellServer.Close()
			apiServerManager.Close()
		}
	}()

	if err := apiServerManager.Run(); err != nil {
		log.Fatal(err)
	}

	if err := shellServer.Wait(); err != nil {
		log.Fatal(err)
	}
}
