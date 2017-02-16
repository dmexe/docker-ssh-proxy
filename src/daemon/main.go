package main

import (
	payload "daemon/payload"
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

	jwtPayload, err := payload.NewJwtParserFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	dockerClient, err := NewDockerClient()
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewSshServer(privateKeyFile, listenAddress, func() (interface{}, error) {
		return NewDockerAgent(dockerClient, jwtPayload)
	})

	if err != nil {
		log.Fatal(err)
	}

	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}
}
