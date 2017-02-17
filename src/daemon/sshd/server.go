package sshd

import (
	"daemon/handlers"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"strings"
)

type CreateServerOptions struct {
	PrivateKeyFile  string
	PrivateKeyBytes []byte
	ListenAddr      string
}

type Server struct {
	config        *ssh.ServerConfig
	listenAddress string
	handlerFunc   handlers.HandlerFunc
	listener      net.Listener
	completed     chan error
}

func NewServer(opts CreateServerOptions, agentCreateFn handlers.HandlerFunc) (*Server, error) {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	if len(opts.PrivateKeyBytes) == 0 {
		privateBytes, err := ioutil.ReadFile(opts.PrivateKeyFile)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to load private key %s (%s)", opts.PrivateKeyFile, err))
		}
		opts.PrivateKeyBytes = privateBytes
	}

	private, err := ssh.ParsePrivateKey(opts.PrivateKeyBytes)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse private key (%s)", err))
	}

	config.AddHostKey(private)

	server := &Server{
		config:        config,
		listenAddress: opts.ListenAddr,
		handlerFunc:   agentCreateFn,
		completed:     make(chan error),
	}

	return server, nil
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) Close() error {
	if s.listener != nil {
		err := s.listener.Close()
		if err != nil {
			return errors.New(fmt.Sprintf("Could not close server listener (%s)", err))
		} else {
			s.listener = nil
		}
	}

	return nil
}

func (s *Server) Wait() error {
	select {
	case err := <-s.completed:
		log.Infof("Server completed")
		return err
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to listen on %s (%s)", s.listenAddress, err))
	}
	s.listener = listener

	log.Printf("Listening on %s...", s.listenAddress)

	go func() {
		defer func() {
			log.Debugf("Stop accepting incoming connections")
			s.completed <- nil
		}()

		for {
			tcpConn, err := listener.Accept()

			if err != nil {
				if strings.HasSuffix(err.Error(), "use of closed network connection") {
					break
				}
				log.Errorf("Failed to accept incoming connection (%s)", err)
				break
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
				handlerFunc: s.handlerFunc,
			}

			log.Infof("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

			if err := clientHandler.Handle(); err != nil {
				log.Errorf("Could not handle client connection (%s)", err)
				s.closeClient(sshConn)
				continue
			}
		}
	}()

	return nil
}

func (s *Server) closeClient(sshConn ssh.Conn) {
	if err := sshConn.Close(); err != nil {
		log.Errorf("Could not handle client connection (%s)", err)
	}
	log.Info("Client connection successfully closed")
}
