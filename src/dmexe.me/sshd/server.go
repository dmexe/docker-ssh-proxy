package sshd

import (
	"context"
	"dmexe.me/payloads"
	"dmexe.me/sshd/handlers"
	"dmexe.me/utils"
	"fmt"
	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"net"
	"strings"
	"sync"
)

// ServerOptions keeps parameters for server instance
type ServerOptions struct {
	PrivateKey  []byte
	Host        string
	Port        uint
	HandlerFunc handlers.HandlerFunc
	Parser      payloads.Parser
}

// Server implements sshd server
type Server struct {
	config        *ssh.ServerConfig
	listenAddress string
	handlerFunc   handlers.HandlerFunc
	listener      net.Listener
	log           *logrus.Entry
	parser        payloads.Parser
	ctx           context.Context
}

// NewServer creates a new sshd server instance using given options
func NewServer(ctx context.Context, opts ServerOptions) (*Server, error) {
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
		listenAddress: fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		handlerFunc:   opts.HandlerFunc,
		parser:        opts.Parser,
		log:           utils.NewLogEntry("ssh.server"),
		ctx:           ctx,
	}

	return server, nil
}

// Addr returns listening addr
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

// Run server
func (s *Server) Run(wg *sync.WaitGroup) error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return fmt.Errorf("Failed to listen on %s (%s)", s.listenAddress, err)
	}
	s.listener = listener

	wg.Add(1)
	go s.loop(wg)
	go s.deadline()

	return nil
}

func (s *Server) deadline() error {
	<-s.ctx.Done()

	s.log.Debug("Context done")

	if err := s.listener.Close(); err != nil {
		s.log.Errorf("Could not close listener (%s)", err)
		return err
	}

	return s.ctx.Err()
}

func (s *Server) loop(wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		s.log.Debugf("Stop accepting incoming connections")
	}()

	s.log.Printf("Listening on %s...", s.listenAddress)

	for {
		tcpConn, err := s.listener.Accept()

		if _, ok := s.ctx.Deadline(); ok {
			s.log.Debug("Context deadline")
			break
		}

		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				s.log.Errorf("Failed to accept incoming connection (%s)", err)
			}
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

		session := NewSession(s.ctx, &SessionOptions{
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
}

func (s *Server) closeSession(sshConn ssh.Conn) {
	if err := sshConn.Close(); err != nil {
		s.log.Errorf("Could not handle client connection (%s)", err)
	}
	s.log.Info("Client connection successfully closed")
}
