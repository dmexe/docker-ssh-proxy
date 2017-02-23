package main

import (
	"context"
	"dmexe.me/apiserver"
	"dmexe.me/apiserver/aggregator"
	"dmexe.me/apiserver/marathon"
	"dmexe.me/payloads"
	"dmexe.me/sshd"
	"dmexe.me/sshd/handlers"
	"dmexe.me/utils"
	"errors"
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type apiMarathonConfig struct {
	urls []string
}

func (m *apiMarathonConfig) description() string {
	return `The Marathon API prefix. This prefix depends on your Marathon configuration.
	For example, running Marathon locally, the API is available at localhost:8080/v2/,
	while the default setup on AWS/DCOS is  $(dcos config show core.dcos_url)/marathon/v2/apps.
	(Can be specified multiple times)`
}

func (m *apiMarathonConfig) String() string {
	return strings.Join(m.urls, " ")
}

func (m *apiMarathonConfig) Set(value string) error {
	m.urls = append(m.urls, value)
	return nil
}

type debugConfig struct {
	token   string
	enabled bool
}

type shellConfig struct {
	host    string
	port    uint
	keyFile string
	enabled bool
}

type apiAggregatorConfig struct {
	interval time.Duration
}

type apiConfig struct {
	host       string
	port       uint
	marathon   apiMarathonConfig
	aggregator apiAggregatorConfig
	enabled    bool
}

type appConfig struct {
	shell shellConfig
	api   apiConfig
	debug debugConfig
	log   *logrus.Entry
	ctx   context.Context
}

func newAppConfig(ctx context.Context) appConfig {
	return appConfig{
		shell: shellConfig{
			host:    "0.0.0.0",
			port:    2200,
			keyFile: "./id_rsa",
		},
		api: apiConfig{
			host:     "0.0.0.0",
			port:     2201,
			marathon: apiMarathonConfig{},
			aggregator: apiAggregatorConfig{
				interval: time.Duration(time.Minute),
			},
		},
		debug: debugConfig{},
		log:   utils.NewLogEntry("config"),
		ctx:   ctx,
	}
}

func (cfg *appConfig) parseArgs() {
	// ssh config
	flag.StringVar(&cfg.shell.host, "ssh.host", cfg.shell.host, "The local addresses ssh should listen on")
	flag.UintVar(&cfg.shell.port, "ssh.port", cfg.shell.port, "The port number that ssh listens on")
	flag.StringVar(&cfg.shell.keyFile, "ssh.key", cfg.shell.keyFile, "The file containing a private host key used by ssh")
	flag.BoolVar(&cfg.shell.enabled, "ssh", cfg.shell.enabled, "Start the ssh server")

	// api server config
	flag.StringVar(&cfg.api.host, "api.host", cfg.api.host, "The local addresses api server should listen on")
	flag.UintVar(&cfg.api.port, "api.port", cfg.api.port, "The port number that api server listens on")
	flag.Var(&cfg.api.marathon, "api.marathon.url", cfg.api.marathon.description())
	flag.DurationVar(&cfg.api.aggregator.interval, "api.interval", cfg.api.aggregator.interval, "The pool interval")
	flag.BoolVar(&cfg.api.enabled, "api", cfg.api.enabled, "Start the api server")

	// debug config
	flag.StringVar(&cfg.debug.token, "debug.token", cfg.debug.token, "The debug token")
	flag.BoolVar(&cfg.debug.enabled, "debug", false, "Enable debug output")

	flag.Parse()
}

func (cfg *appConfig) validate() error {
	if cfg.api.enabled {
		if len(cfg.api.marathon.urls) == 0 {
			return errors.New("API server enabled, but no urls specified, please add at least one [-api.marathon.url] flag")
		}
	}

	if !cfg.api.enabled && !cfg.shell.enabled {
		return errors.New("No listeners, please add at least one of this flags [-ssh] [-api]")
	}

	return nil
}

func (cfg *appConfig) getPayloadParser() payloads.Parser {
	if cfg.debug.token != "" {
		cfg.log.Warnf("Force payload to ContainerID=%s", cfg.debug.token)
		return &payloads.EchoParser{
			Payload: payloads.Payload{ContainerID: cfg.debug.token},
		}
	}

	jwtParser, err := payloads.NewJwtParserFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return jwtParser
}

func (cfg *appConfig) getDockerClient() *docker.Client {
	dockerClient, err := handlers.NewDockerClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	return dockerClient
}

func (cfg *appConfig) getDockerShellHandler(dockerClient *docker.Client) handlers.HandlerFunc {
	handler := func() (handlers.Handler, error) {
		return handlers.NewDockerHandler(handlers.DockerHandlerOptions{
			Client: dockerClient,
		})
	}
	return handler
}

func (cfg *appConfig) getPrivateKey() []byte {
	privateKey, err := ioutil.ReadFile(cfg.shell.keyFile)
	if err != nil {
		log.Fatal(err)
	}
	return privateKey
}

func (cfg *appConfig) getShellServer(privateKey []byte, handlerFunc handlers.HandlerFunc, payloadParser payloads.Parser) *sshd.Server {
	serverOptions := sshd.ServerOptions{
		PrivateKey:  privateKey,
		Host:        cfg.shell.host,
		Port:        cfg.shell.port,
		HandlerFunc: handlerFunc,
		Parser:      payloadParser,
	}

	server, err := sshd.NewServer(cfg.ctx, serverOptions)
	if err != nil {
		log.Fatal(err)
	}

	return server
}

func (cfg *appConfig) getAPIProvider() apiserver.RunnableProvider {
	providers := make([]apiserver.Provider, 0)

	for _, url := range cfg.api.marathon.urls {
		providerOptions := marathon.ProviderOptions{
			Endpoint: url,
		}
		provider, err := marathon.NewProvider(providerOptions)
		if err != nil {
			log.Fatal(err)
		}
		providers = append(providers, provider)
	}

	aggregatorOptions := aggregator.ProviderOptions{
		Providers: providers,
		Interval:  cfg.api.aggregator.interval,
	}

	manager, err := aggregator.NewProvider(cfg.ctx, aggregatorOptions)
	if err != nil {
		log.Fatal(err)
	}

	return manager
}

func (cfg *appConfig) getAPIServer(provider apiserver.Provider) *apiserver.Server {
	opts := apiserver.ServerOptions{
		Host:     cfg.api.host,
		Port:     cfg.api.port,
		Provider: provider,
	}

	apiServer, err := apiserver.NewServer(cfg.ctx, opts)
	if err != nil {
		cfg.log.Fatal(err)
	}

	return apiServer
}
