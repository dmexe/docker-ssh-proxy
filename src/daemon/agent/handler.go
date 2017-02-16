package agent

import (
	"io"
)

type ResizeRequest struct {
	Width  uint32
	Height uint32
}

type TtyRequest struct {
	Term   string
	Width  uint32
	Height uint32
}

type HandleRequest struct {
	Payload string
	Tty     *TtyRequest
	Reader  io.Reader
	Writer  io.Writer
}

type Handler interface {
	Handle(req *HandleRequest) error
	IsHandled() bool
	Wait() error
	Resize(tty *ResizeRequest) error
	Close() error
}

type CreateHandler func() (Handler, error)

func (req *TtyRequest) Resize() *ResizeRequest {
	return &ResizeRequest{
		Width:  req.Width,
		Height: req.Height,
	}
}
