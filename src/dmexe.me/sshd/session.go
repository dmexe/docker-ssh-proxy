package sshd

import (
	"context"
	"dmexe.me/payloads"
	"dmexe.me/sshd/handlers"
	"dmexe.me/utils"
	"fmt"
	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"sync"
)

// SessionOptions keeps parameters for constructor
type SessionOptions struct {
	Conn        *ssh.ServerConn
	NewChannels <-chan ssh.NewChannel
	Requests    <-chan *ssh.Request
	HandlerFunc handlers.HandlerFunc
	Payload     payloads.Payload
}

// Session uses for handing ssh client requests
type Session struct {
	sync.Mutex
	conn        *ssh.ServerConn
	newChannels <-chan ssh.NewChannel
	requests    <-chan *ssh.Request
	handlerFunc handlers.HandlerFunc
	handlerTty  *handlers.Tty
	handler     handlers.Handler
	log         *logrus.Entry
	payload     payloads.Payload
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewSession creates a new consumer for incoming ssh connection
func NewSession(ctx context.Context, options *SessionOptions) *Session {
	ctx, cancel := context.WithCancel(ctx)

	session := &Session{
		conn:        options.Conn,
		newChannels: options.NewChannels,
		requests:    options.Requests,
		handlerFunc: options.HandlerFunc,
		payload:     options.Payload,
		log:         utils.NewLogEntry("ssh.session"),
		ctx:         ctx,
		cancel:      cancel,
	}
	return session
}

// Handle new client request
func (s *Session) Handle() error {
	go func() {
		for newChannel := range s.newChannels {
			s.handleChannelRequest(newChannel)
		}
	}()

	go func() {
		for req := range s.requests {
			reqReply(req, false, s.log)
		}
	}()

	return nil
}

func (s *Session) handleChannelRequest(newChannel ssh.NewChannel) {
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		s.log.Warnf("Unknown requested channel type: %s", t)
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		s.log.Errorf("Could not accept channel (%s)", err)
		return
	}

	go func() {

		for {
			select {
			case <-s.ctx.Done():
				s.log.Debug("Context done")
				s.closeChannel(channel)
				return

			case req := <-requests:

				if req == nil {
					s.log.Debug("Client closed connection")
					return
				}

				switch req.Type {

				case "exec", "shell":
					s.handleCommandReq(req, channel)

				case "pty-req":
					s.handleTtyReq(req)

				case "window-change":
					s.handleResizeReq(req)

				default:
					reqReply(req, false, s.log)
				}
			}
		}
	}()
}

func (s *Session) handleResizeReq(req *ssh.Request) {
	if !s.isTTY() {
		s.log.Warn("'window-change' request called before 'tty-req' request")
		reqReply(req, false, s.log)
		return
	}

	if !s.isHandled() {
		s.log.Warn("'window-changed' request called without 'exec' request")
		reqReply(req, false, s.log)
		return
	}

	resize, err := reqParseWinchPayload(req.Payload)
	if err != nil {
		s.log.Errorf("Could not parse 'window-change' request (%s)", err)
		reqReply(req, false, s.log)
		return
	}

	if err := s.handler.Resize(resize); err != nil {
		s.log.Errorf("Could not handle 'window-change' request (%s)", err)
		reqReply(req, false, s.log)
		return
	}

	reqReply(req, true, s.log)
}

func (s *Session) handleCommandReq(req *ssh.Request, channel ssh.Channel) {
	if s.isHandled() {
		s.log.Warn("'exec' request called multiple times")
		reqReply(req, false, s.log)
		return
	}

	handleRequest := &handlers.Request{
		Tty:     s.handlerTty,
		Stdin:   channel.(io.Reader),
		Stdout:  channel.(io.Writer),
		Stderr:  channel.Stderr(),
		Payload: s.payload,
	}

	if req.Type == "exec" {
		execReq, err := reqParseExecPayload(req.Payload)
		if err != nil {
			s.log.Errorf("Could not parse request payloads (%s)", err)
			reqReply(req, false, s.log)
			return
		}
		handleRequest.Exec = string(execReq)
	}

	sessionHandler, err := s.handlerFunc()
	if err != nil {
		s.log.Errorf("Could not create a new handler (%s)", err)
		reqReply(req, false, s.log)
		return
	}

	s.setHandler(sessionHandler)

	go func() {
		resp, err := sessionHandler.Handle(s.ctx, handleRequest)
		if err != nil {
			s.log.Errorf("Could not handle request (%s)", err)
		}
		s.sendExitReply(channel, uint32(resp.Code))
		s.cancel()
	}()

	reqReply(req, true, s.log)

	s.log.Debugf("Request handled")
}

func (s *Session) handleTtyReq(req *ssh.Request) {
	if s.isTTY() {
		s.log.Warnf("'tty-req' request called multiple times")
		return
	}

	tty, err := reqParseTtyPayload(req.Payload)
	if err != nil {
		s.log.Error(err)
		reqReply(req, false, s.log)
		return
	}

	s.setTTY(tty)
	reqReply(req, true, s.log)
}

func (s *Session) sendExitReply(channel ssh.Channel, code uint32) {
	if _, err := channel.SendRequest("exit-status", false, buildExitStatus(code)); err != nil {
		s.log.Warnf("Could not send 'exit-status' request (%s)", err)
	} else {
		s.log.Debugf("Sent request 'exit-status' (%d)", code)
	}
}

func (s *Session) closeChannel(channel ssh.Channel) {
	if s.isHandled() {
		if err := s.handler.Close(); err != nil {
			s.log.Errorf("Could not close handlers (%s)", err)
		}
	}

	if err := channel.Close(); err != nil {
		if err.Error() != "EOF" {
			s.log.Warnf("Could not close channel (%s)", err)
		} else {
			s.log.Debugf("Could not close channel (%s)", err)
		}
	} else {
		s.log.Debug("Channel closed")
	}
}

func (s *Session) isHandled() bool {
	s.Lock()
	defer s.Unlock()
	return s.handler != nil
}

func (s *Session) setHandler(handler handlers.Handler) {
	s.Lock()
	defer s.Unlock()
	s.handler = handler
}

func (s *Session) isTTY() bool {
	s.Lock()
	defer s.Unlock()
	return s.handlerTty != nil
}

func (s *Session) setTTY(tty *handlers.Tty) {
	s.Lock()
	defer s.Unlock()
	s.handlerTty = tty
}
