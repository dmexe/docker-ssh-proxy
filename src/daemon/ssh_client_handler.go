package main

import (
	"bytes"
	"daemon/agent"
	"encoding/binary"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"sync"
	"time"
)

type SshClientHandler struct {
	sshConn     *ssh.ServerConn
	newChannels <-chan ssh.NewChannel
	requests    <-chan *ssh.Request
	createAgent agent.CreateHandler
}

func (h *SshClientHandler) Handle() error {
	go h.handleConnectionRequests()
	go h.handleChannelRequests()
	return nil
}

func (h *SshClientHandler) handleChannelRequests() {
	for newChannel := range h.newChannels {
		h.handleChannelRequest(newChannel)
	}
}

func (h *SshClientHandler) handleConnectionRequests() {
	for req := range h.requests {
		h.replyReq(req, false)
	}
}

func (h *SshClientHandler) handleChannelRequest(newChannel ssh.NewChannel) {
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		log.Warnf("Unknown requested channel type: %s", t)
		return
	}

	connection, requests, err := newChannel.Accept()
	if err != nil {
		log.Errorf("Could not accept channel (%s)", err)
		return
	}

	agentHandler, err := h.createAgent()
	if err != nil {
		h.closeChannel(connection)
		log.Errorf("Could not create agent (%s)", err)
		return
	}

	closer := h.createChannelCloser(agentHandler, connection)

	go func() {
		defer closer()

		var agentTty *agent.TtyRequest

		for {
			select {

			case <-time.After(1 * time.Second):
				if !agentHandler.IsHandled() {
					log.Warn("Could not handle request within 1 second")
					goto END_LOOP
				}

			case req := <-requests:
				if req == nil {
					goto END_LOOP
				}

				switch req.Type {

				case "exec":
					payload, err := h.parseExecReq(req.Payload)
					if err != nil {
						h.replyReq(req, false)
						continue
					}

					handleRequest := &agent.HandleRequest{
						Payload: string(payload),
						Tty:     agentTty,
						Reader:  connection.(io.Reader),
						Writer:  connection.(io.Writer),
					}

					if err := agentHandler.Handle(handleRequest); err != nil {
						log.Error(err)
						h.replyReq(req, false)
						continue
					}

					h.replyReq(req, true)

					go func() {
						defer closer()
						if err := agentHandler.Wait(); err != nil {
							log.Error(err)
						}
					}()

				case "shell":
					h.replyReq(req, true)

				case "pty-req":
					tty, err := h.parsePtyReq(req.Payload)
					if err != nil {
						log.Error(err)
						h.replyReq(req, false)
						continue
					}
					agentTty = tty
					h.replyReq(req, true)

				case "window-change":
					resize, err := h.parseDims(req.Payload)
					if err != nil {
						log.Error(err)
						h.replyReq(req, false)
						continue
					}

					if err := agentHandler.Resize(resize); err != nil {
						log.Error(err)
						h.replyReq(req, false)
						continue
					}

					h.replyReq(req, true)

				default:
					h.replyReq(req, false)
				}
			}
		}

	END_LOOP:
	}()
}

func (h *SshClientHandler) parseExecReq(b []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(b)
	execLenBytes := buffer.Next(4)
	if len(execLenBytes) != 4 {
		return nil, errors.New(fmt.Sprintf("Could not read pty-req, expected len=4, got %d", len(execLenBytes)))
	}

	execLen := binary.BigEndian.Uint32(execLenBytes)

	execBytes := buffer.Next(int(execLen))
	if len(execBytes) != int(execLen) {
		return nil, errors.New(fmt.Sprintf("Could not read TERM, expected len=%d, got %d", execLenBytes, len(execBytes)))
	}

	return execBytes, nil
}

func (h *SshClientHandler) parsePtyReq(b []byte) (*agent.TtyRequest, error) {
	buffer := bytes.NewBuffer(b)
	termLenBytes := buffer.Next(4)
	if len(termLenBytes) != 4 {
		return nil, errors.New(fmt.Sprintf("Could not read pty-req, expected len=4, got %d", len(termLenBytes)))
	}

	termLen := binary.BigEndian.Uint32(termLenBytes)

	termBytes := buffer.Next(int(termLen))
	if len(termBytes) != int(termLen) {
		return nil, errors.New(fmt.Sprintf("Could not read TERM, expected len=%d, got %d", termLen, len(termBytes)))
	}

	dimsBytes := buffer.Next(8)
	if len(dimsBytes) != 8 {
		return nil, errors.New(fmt.Sprintf("Could not read demissins, expected len=8, got %d", len(dimsBytes)))
	}

	resize, err := h.parseDims(dimsBytes)
	if err != nil {
		return nil, err
	}

	req := &agent.TtyRequest{
		Term:   string(termBytes),
		Width:  resize.Width,
		Height: resize.Height,
	}

	return req, nil
}

func (h *SshClientHandler) replyReq(req *ssh.Request, value bool) {
	if req.WantReply {
		if err := req.Reply(value, nil); err != nil {
			log.Warnf("Could not send reply %s (%s)", req.Type, err)
		} else {
			log.Debugf("Send reply %s (%t)", req.Type, value)
		}
	} else {
		log.Debugf("Reply ignored %s (%t)", req.Type, value)
	}
}

func (h *SshClientHandler) parseDims(b []byte) (*agent.ResizeRequest, error) {
	if len(b) < 8 {
		return nil, errors.New(fmt.Sprintf("Could not read req demissions, expected buffer len >= 8, got=%d", len(b)))
	}

	width := binary.BigEndian.Uint32(b)
	height := binary.BigEndian.Uint32(b[4:])

	req := &agent.ResizeRequest{
		Width:  width,
		Height: height,
	}

	return req, nil
}

func (h *SshClientHandler) closeChannel(channel ssh.Channel) {
	if err := channel.Close(); err != nil {
		log.Warnf("Could not close channel (%s)", err)
	} else {
		log.Infof("Channel successfuly closed")
	}
}

func (h *SshClientHandler) createChannelCloser(agentHandler agent.Handler, channel ssh.Channel) func() {
	var once sync.Once

	closeHandler := func() {
		if err := agentHandler.Close(); err != nil {
			log.Errorf("Could not close agent (%s)", err)
		} else {
			log.Debugf("Agent successfuly closed")
		}

		h.closeChannel(channel)
	}

	return func() {
		once.Do(closeHandler)
	}
}

func (h *SshClientHandler) CloseConn() error {
	if h.sshConn != nil {
		err := h.sshConn.Close()
		if err != nil {
			log.Warnf("Could not close connection (%s)", err)
			return err
		} else {
			log.Infof("Connection successfuly closed")
			h.sshConn = nil
		}
	}
	return nil
}
