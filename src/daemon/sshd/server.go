package sshd

import (
	"daemon/payloads"
	"daemon/sshd/handlers"
	"daemon/utils"
	"fmt"
	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"net"
)

// ServerOptions keeps parameters for server instance
type ServerOptions struct {
	PrivateKey  []byte
	ListenAddr  string
	HandlerFunc handlers.HandlerFunc
	Parser      payloads.Parser
}

// Server implements sshd server
type Server struct {
	config        *ssh.ServerConfig
	listenAddress string
	handlerFunc   handlers.HandlerFunc
	listener      net.Listener
	completed     chan error
	closed        bool
	log           *logrus.Entry
	parser        payloads.Parser
}

// NewServer creates a new sshd server instance using given options and session handlers constructor
func NewServer(opts ServerOptions) (*Server, error) {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	private, err := ssh.ParsePrivateKey(opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key (%s)", err)
	}

	config.AddHostKey(private)

	server := &Server{
		config:        config,
		listenAddress: opts.ListenAddr,
		handlerFunc:   opts.HandlerFunc,
		parser:        opts.Parser,
		completed:     make(chan error, 1),
		log:           utils.NewLogEntry("sshd.server"),
	}

	return server, nil
}

// Addr returns listening addr
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

// Close server listener
func (s *Server) Close() error {
	if s.closed {
		s.log.Warnf("Server close called multiple times")
	}
	s.closed = true

	err := s.listener.Close()
	if err != nil {
		return fmt.Errorf("Could not close server listener (%s)", err)
	}

	return nil
}

// Wait until server stops
func (s *Server) Wait() error {
	select {
	case err := <-s.completed:
		s.log.Infof("Server completed")
		return err
	}
}

// Run server
func (s *Server) Run() error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return fmt.Errorf("Failed to listen on %s (%s)", s.listenAddress, err)
	}
	s.listener = listener

	s.log.Printf("Listening on %s...", s.listenAddress)

	go func() {
		defer func() {
			s.log.Debugf("Stop accepting incoming connections")
			s.completed <- nil
		}()

		for {
			tcpConn, err := listener.Accept()

			if s.closed {
				break
			}

			if err != nil {
				s.log.Errorf("Failed to accept incoming connection (%s)", err)
				break
			}

			sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, s.config)
			if err != nil {
				s.log.Errorf("Failed to handshake (%s)", err)
				continue
			}

			s.log.Infof("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

			payload, err := s.parser.Parse(sshConn.User())
			if err != nil {
				s.log.Warnf("Could not parse payload (%s)", err)
				s.closeSession(sshConn)
				continue
			}

			session := NewSession(&SessionOptions{
				Conn:        sshConn,
				NewChannels: chans,
				Requests:    reqs,
				HandlerFunc: s.handlerFunc,
				Payload:     payload,
			})

			if err := session.Handle(); err != nil {
				s.log.Errorf("Could not handle client connection (%s)", err)
				s.closeSession(sshConn)
				continue
			}
		}
	}()

	return nil
}

func (s *Server) closeSession(sshConn ssh.Conn) {
	if err := sshConn.Close(); err != nil {
		s.log.Errorf("Could not handle client connection (%s)", err)
	}
	s.log.Info("Client connection successfully closed")
}
