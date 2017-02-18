package sshd

import (
	"daemon/handlers"
	"daemon/utils"
	"fmt"
	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"strings"
)

// CreateServerOptions keeps parameters for server instance
type CreateServerOptions struct {
	PrivateKeyFile  string
	PrivateKeyBytes []byte
	ListenAddr      string
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
}

// NewServer creates a new sshd server instance using given options and session handlers constructor
func NewServer(opts CreateServerOptions, handlerFurn handlers.HandlerFunc) (*Server, error) {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	if len(opts.PrivateKeyBytes) == 0 {
		privateBytes, err := ioutil.ReadFile(opts.PrivateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to load private key %s (%s)", opts.PrivateKeyFile, err)
		}
		opts.PrivateKeyBytes = privateBytes
	}

	private, err := ssh.ParsePrivateKey(opts.PrivateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key (%s)", err)
	}

	config.AddHostKey(private)

	server := &Server{
		config:        config,
		listenAddress: opts.ListenAddr,
		handlerFunc:   handlerFurn,
		completed:     make(chan error),
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

// Start server
func (s *Server) Start() error {
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

			if err != nil {
				if strings.HasSuffix(err.Error(), "use of closed network connection") {
					break
				}
				s.log.Errorf("Failed to accept incoming connection (%s)", err)
				break
			}

			sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, s.config)
			if err != nil {
				s.log.Errorf("Failed to handshake (%s)", err)
				continue
			}

			s.log.Infof("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

			session := NewSession(&CreateSessionOptions{
				Conn:        sshConn,
				NewChannels: chans,
				Requests:    reqs,
				HandlerFunc: s.handlerFunc,
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
