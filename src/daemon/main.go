package main

import (
	"daemon/agent"
	"daemon/payload"
	"daemon/sshd"
	"flag"
	log "github.com/Sirupsen/logrus"
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

	server, err := sshd.NewServer(privateKeyFile, listenAddress, func(payload string) (agent.Handler, error) {
		filter, err := jwtParser.Parse(payload)
		if err != nil {
			return nil, err
		}
		return agent.NewDockerHandler(dockerClient, filter)
	})

	if err != nil {
		log.Fatal(err)
	}

	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}
}
