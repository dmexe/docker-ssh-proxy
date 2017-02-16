package main

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
)

type SshServer struct {
	config        *ssh.ServerConfig
	listenAddress string
	agentCreateFn AgentCreateFunc
}

func NewSshServer(privateKeyFile string, listenAddress string, agentCreateFn AgentCreateFunc) (*SshServer, error) {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	privateBytes, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to load private key %s (%s)", privateKeyFile, err))
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse private key (%s)", err))
	}

	config.AddHostKey(private)

	server := &SshServer{
		config:        config,
		listenAddress: listenAddress,
		agentCreateFn: agentCreateFn,
	}

	return server, nil
}

func (srv *SshServer) Start() error {
	listener, err := net.Listen("tcp", srv.listenAddress)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to listen on %s (%s)", srv.listenAddress, err))
	}

	log.Printf("Listening on %s...", srv.listenAddress)

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Errorf("Failed to accept incoming connection (%s)", err)
			continue
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, srv.config)
		if err != nil {
			log.Errorf("Failed to handshake (%s)", err)
			continue
		}

		clientHandler := &SshClientHandler{
			sshConn:       sshConn,
			newChannels:   chans,
			requests:      reqs,
			agentCreateFn: srv.agentCreateFn,
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

		if err := clientHandler.Handle(); err != nil {
			log.Errorf("Could not handle client requests (%s)", err)
			clientHandler.CloseConn()
			continue
		}
	}
}
