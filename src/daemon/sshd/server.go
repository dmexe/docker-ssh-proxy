package sshd

import (
	"daemon/agent"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
)

type Server struct {
	config        *ssh.ServerConfig
	listenAddress string
	createAgent   agent.CreateHandler
}

func NewServer(privateKeyFile string, listenAddress string, agentCreateFn agent.CreateHandler) (*Server, error) {
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

	server := &Server{
		config:        config,
		listenAddress: listenAddress,
		createAgent:   agentCreateFn,
	}

	return server, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to listen on %s (%s)", s.listenAddress, err))
	}

	log.Printf("Listening on %s...", s.listenAddress)

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Errorf("Failed to accept incoming connection (%s)", err)
			continue
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, s.config)
		if err != nil {
			log.Errorf("Failed to handshake (%s)", err)
			continue
		}

		clientHandler := &Session{
			conn:        sshConn,
			newChannels: chans,
			requests:    reqs,
			createAgent: s.createAgent,
		}

		log.Infof("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

		if err := clientHandler.Handle(); err != nil {
			log.Errorf("Could not handle client connection (%s)", err)
			s.closeClient(sshConn)
			continue
		}
	}
}

func (s *Server) closeClient(sshConn ssh.Conn) {
	if err := sshConn.Close(); err != nil {
		log.Errorf("Could not handle client connection (%s)", err)
	}
	log.Info("Client connection successfully closed")
}
