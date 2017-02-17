package sshd

import (
	"golang.org/x/crypto/ssh"
	log "github.com/Sirupsen/logrus"
	"bytes"
	"fmt"
	"errors"
	"encoding/binary"
	"daemon/agent"
)

func reqReply(req *ssh.Request, value bool) {
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

func reqParseExecPayload(b []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(b)
	execLenBytes := buffer.Next(4)
	if len(execLenBytes) != 4 {
		return nil, errors.New(fmt.Sprintf("Could not read 'exec' request, expected len=4, got %d", len(execLenBytes)))
	}

	execLen := binary.BigEndian.Uint32(execLenBytes)
	execBytes := buffer.Next(int(execLen))
	if len(execBytes) != int(execLen) {
		return nil, errors.New(fmt.Sprintf("Could not read TERM, expected len=%d, got %d", execLenBytes, len(execBytes)))
	}

	return execBytes, nil
}

func reqParseWinchPayload(b []byte) (*agent.ResizeRequest, error) {
	if len(b) < 8 {
		return nil, errors.New(
			fmt.Sprintf("Could not read 'window-change' request, expected buffer len >= 8, got=%d",
				len(b),
		))
	}

	width := binary.BigEndian.Uint32(b)
	height := binary.BigEndian.Uint32(b[4:])

	req := &agent.ResizeRequest{
		Width:  width,
		Height: height,
	}

	return req, nil
}

func reqParseTtyPayload(b []byte) (*agent.TtyRequest, error) {
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

	winchBytes := buffer.Next(8)
	if len(winchBytes) != 8 {
		return nil, errors.New(fmt.Sprintf("Could not read demissins, expected len=8, got %d", len(winchBytes)))
	}

	resize, err := reqParseWinchPayload(winchBytes)
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
