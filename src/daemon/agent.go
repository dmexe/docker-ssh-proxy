package main

import (
	"io"
)

type AgentResizeRequest struct {
	Width  uint32
	Height uint32
}

type AgentTtyRequest struct {
	Term   string
	Width  uint32
	Height uint32
}

type AgentHandleRequest struct {
	Payload string
	Tty     *AgentTtyRequest
	Reader  io.Reader
	Writer  io.Writer
}

type Agent interface {
	Handle(req *AgentHandleRequest) error
	IsHandled() bool
	Wait() error
	Resize(tty *AgentResizeRequest) error
	Close() error
}

type AgentCreateFunc func() (interface{}, error)

func (req *AgentTtyRequest) Resize() *AgentResizeRequest {
	return &AgentResizeRequest{
		Width:  req.Width,
		Height: req.Height,
	}
}
