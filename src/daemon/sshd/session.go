package sshd

import (
	"daemon/handlers"
	"fmt"
	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"time"
)

// CreateSessionOptions keeps parameters for constructor
type CreateSessionOptions struct {
	Conn        *ssh.ServerConn
	NewChannels <-chan ssh.NewChannel
	Requests    <-chan *ssh.Request
	HandlerFunc handlers.HandlerFunc
	Log         *logrus.Entry
}

// Session uses for handing ssh client requests
type Session struct {
	conn        *ssh.ServerConn
	newChannels <-chan ssh.NewChannel
	requests    <-chan *ssh.Request
	handlerFunc handlers.HandlerFunc
	handlerTty  *handlers.Tty
	handler     handlers.Handler
	handled     bool
	exited      bool
	closed      bool
	log         *logrus.Entry
}

// NewSession creates a new consumer for incoming ssh connection
func NewSession(options *CreateSessionOptions) *Session {
	session := &Session{
		conn:        options.Conn,
		newChannels: options.NewChannels,
		requests:    options.Requests,
		handlerFunc: options.HandlerFunc,
		log:         options.Log.WithField("logger", "sshd.session"),
	}
	return session
}

// Handle new client request
func (s *Session) Handle() error {
	go s.handleConnectionRequests()
	go s.handleChannelRequests()
	return nil
}

func (s *Session) handleChannelRequests() {
	for newChannel := range s.newChannels {
		s.handleChannelRequest(newChannel)
	}
}

func (s *Session) handleConnectionRequests() {
	for req := range s.requests {
		reqReply(req, false)
	}
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
		defer s.closeChannel(channel)

		for {
			select {

			case <-time.After(10 * time.Second):
				if !s.handled {
					s.log.Warn("Could not handle request within 10 second")
					goto END_LOOP
				}

			case req := <-requests:
				if req == nil {
					s.log.Debug("No more requests")
					goto END_LOOP
				}

				switch req.Type {

				case "exec", "shell":
					s.handleCommandReq(req, channel)

				case "pty-req":
					s.handleTtyReq(req)

				case "window-change":
					s.handleResizeReq(req)

				default:
					reqReply(req, false)
				}
			}
		}

	END_LOOP:
	}()
}

func (s *Session) handleResizeReq(req *ssh.Request) {
	if s.handlerTty == nil {
		s.log.Warn("'window-change' request called before 'tty-req' request")
		reqReply(req, false)
		return
	}

	if s.handler == nil {
		s.log.Warn("'window-changed' request called without 'exec' request")
		reqReply(req, false)
		return
	}

	resize, err := reqParseWinchPayload(req.Payload)
	if err != nil {
		s.log.Errorf("Could not parse 'window-change' request (%s)", err)
		reqReply(req, false)
		return
	}

	if err := s.handler.Resize(resize); err != nil {
		s.log.Errorf("Could not handle 'window-change' request (%s)", err)
		reqReply(req, false)
		return
	}

	reqReply(req, true)
}

func (s *Session) handleCommandReq(req *ssh.Request, channel ssh.Channel) {
	if s.handled {
		s.log.Warn("'exec' request called multiple times")
		reqReply(req, false)
		return
	}
	s.handled = true

	handleRequest := &handlers.Request{
		Tty:    s.handlerTty,
		Stdin:  channel.(io.Reader),
		Stdout: channel.(io.Writer),
		Stderr: channel.Stderr(),
	}

	if req.Type == "exec" {
		execReq, err := reqParseExecPayload(req.Payload)
		if err != nil {
			s.log.Errorf("Could not parse request payloads (%s)", err)
			reqReply(req, false)
			return
		}
		handleRequest.Exec = string(execReq)
	}

	sessionHandler, err := s.handlerFunc(s.conn.User())
	if err != nil {
		s.log.Errorf("Could not create handlers (%s)", err)
		reqReply(req, false)
		return
	}

	if err := sessionHandler.Handle(handleRequest); err != nil {
		s.log.Errorf("Could not handle request (%s)", err)
		reqReply(req, false)
		return
	}

	s.handler = sessionHandler
	reqReply(req, true)

	s.log.Debugf("Request successfully handled")

	go func() {
		resp, err := sessionHandler.Wait()
		if err != nil {
			s.log.Errorf("Could not wait handler (%s)", err)
		}
		s.exitChannel(channel, uint32(resp.Code))
	}()

}

func (s *Session) handleTtyReq(req *ssh.Request) {
	if s.handlerTty != nil {
		s.log.Warnf("'tty-req' request called multiple times")
		return
	}

	tty, err := reqParseTtyPayload(req.Payload)
	if err != nil {
		s.log.Error(err)
		reqReply(req, false)
		return
	}

	s.handlerTty = tty
	reqReply(req, true)
}

func (s *Session) exitChannel(channel ssh.Channel, code uint32) {
	if s.exited {
		s.log.Warnf("Channel exit called multiple times")
		return
	}
	s.exited = true

	if _, err := channel.SendRequest("exit-status", false, buildExitStatus(code)); err != nil {
		s.log.Warnf("Could not send 'exit-status' request (%s)", err)
		return
	}

	s.log.Debugf("Successfuly send request 'exit-status' (%d)", code)
	s.closeChannel(channel)
}

func (s *Session) closeChannel(channel ssh.Channel) {
	if s.closed {
		s.log.Warnf("Channel close called multiple times")
		return
	}
	s.closed = true

	if s.handler != nil {
		if err := s.handler.Close(); err != nil {
			s.log.Errorf("Could not close handlers (%s)", err)
		} else {
			s.log.Debugf("Agent successfuly closed")
		}
	}

	if err := channel.Close(); err != nil {
		if err.Error() != "EOF" {
			s.log.Warnf("Could not close channel (%s)", err)
		} else {
			s.log.Debugf("Could not close channel (%s)", err)
		}
	} else {
		s.log.Infof("Channel successfuly closed")
	}
}
