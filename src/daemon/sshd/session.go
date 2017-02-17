package sshd

import (
	"daemon/handlers"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"time"
)

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
}

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
		log.Warnf("Unknown requested channel type: %s", t)
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Errorf("Could not accept channel (%s)", err)
		return
	}

	go func() {
		defer s.closeChannel(channel)

		for {
			select {

			case <-time.After(10 * time.Second):
				if !s.handled {
					log.Warn("Could not handle request within 10 second")
					goto END_LOOP
				}

			case req := <-requests:
				if req == nil {
					log.Debug("No more requests")
					goto END_LOOP
				}

				switch req.Type {

				case "exec", "shell":
					s.handleAgentReq(req, channel)

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
		log.Warn("'window-change' request called before 'tty-req' request")
		reqReply(req, false)
		return
	}

	if s.handler == nil {
		log.Warn("'window-changed' request called without 'exec' request")
		reqReply(req, false)
		return
	}

	resize, err := reqParseWinchPayload(req.Payload)
	if err != nil {
		log.Errorf("Could not parse 'window-change' request (%s)", err)
		reqReply(req, false)
		return
	}

	if err := s.handler.Resize(resize); err != nil {
		log.Errorf("Could not handle 'window-change' request (%s)", err)
		reqReply(req, false)
		return
	}

	reqReply(req, true)
}

func (s *Session) handleAgentReq(req *ssh.Request, channel ssh.Channel) {
	if s.handled {
		log.Warn("'exec' request called multiple times")
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
			log.Errorf("Could not parse request payload (%s)", err)
			reqReply(req, false)
			return
		}
		handleRequest.Exec = string(execReq)
	}

	sessionHandler, err := s.handlerFunc(s.conn.User())
	if err != nil {
		log.Errorf("Could not create handlers (%s)", err)
		reqReply(req, false)
		return
	}

	if err := sessionHandler.Handle(handleRequest); err != nil {
		log.Errorf("Could not handle request (%s)", err)
		reqReply(req, false)
		return
	}

	s.handler = sessionHandler
	reqReply(req, true)

	log.Debugf("Request successfully handled")

	go func() {
		code, err := sessionHandler.Wait()
		if err != nil {
			log.Errorf("Could not wait handler (%s)", err)
		}
		s.exitChannel(channel, uint32(code))
	}()

}

func (s *Session) handleTtyReq(req *ssh.Request) {
	if s.handlerTty != nil {
		log.Warnf("'tty-req' request called multiple times")
		return
	}

	tty, err := reqParseTtyPayload(req.Payload)
	if err != nil {
		log.Error(err)
		reqReply(req, false)
		return
	}

	s.handlerTty = tty
	reqReply(req, true)
}

func (s *Session) exitChannel(channel ssh.Channel, code uint32) {
	if s.exited {
		log.Warnf("Channel exit called multiple times")
		return
	}
	s.exited = true

	if _, err := channel.SendRequest("exit-status", false, buildExitStatus(code)); err != nil {
		log.Warnf("Could not send 'exit-status' request (%s)", err)
		return
	} else {
		log.Debugf("Successfuly send request 'exit-status' (%d)", code)
	}

	s.closeChannel(channel)
}

func (s *Session) closeChannel(channel ssh.Channel) {
	if s.closed {
		log.Warnf("Channel close called multiple times")
		return
	}
	s.closed = true

	if s.handler != nil {
		if err := s.handler.Close(); err != nil {
			log.Errorf("Could not close handlers (%s)", err)
		} else {
			log.Debugf("Agent successfuly closed")
		}
	}

	if err := channel.Close(); err != nil {
		if err.Error() != "EOF" {
			log.Warnf("Could not close channel (%s)", err)
		} else {
			log.Debugf("Could not close channel (%s)", err)
		}
	} else {
		log.Infof("Channel successfuly closed")
	}
}
