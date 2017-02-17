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
	Tty    *TtyRequest
	Reader io.Reader
	Writer io.Writer
	Exec   string
}

type Handler interface {
	Handle(req *HandleRequest) error
	Resize(tty *ResizeRequest) error
	Wait() error
	Close() error
}

type CreateHandler func(payload string) (Handler, error)

func (req *TtyRequest) Resize() *ResizeRequest {
	return &ResizeRequest{
		Width:  req.Width,
		Height: req.Height,
	}
}
