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
	flag.StringVar(&privateKeyFile, "ssh.key", "./id_rsa", "host private key file")
	flag.StringVar(&listenAddress, "ssh.listen", "0.0.0.0:2200", "listen address")
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

func getDockerShellHandler(dockerClient *docker.Client) handlers.HandlerFunc {
	handler := func() (handlers.Handler, error) {
		return handlers.NewDockerHandler(handlers.DockerHandlerOptions{
			Client: dockerClient,
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

func getShellServer(privateKey []byte, handlerFunc handlers.HandlerFunc, payloadParser payloads.Parser) *sshd.Server {
	serverOptions := sshd.ServerOptions{
		PrivateKey:  privateKey,
		ListenAddr:  listenAddress,
		HandlerFunc: handlerFunc,
		Parser:      payloadParser,
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
	dockerShellHandler := getDockerShellHandler(dockerClient)
	privateKey := getPrivateKey()
	shellServer := getShellServer(privateKey, dockerShellHandler, payloadParser)

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
