package handlers

import (
	"io"
)

type Resize struct {
	Width  uint32
	Height uint32
}

type Tty struct {
	Term   string
	Width  uint32
	Height uint32
}

type Request struct {
	Tty    *Tty
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Exec   string
}

type Response struct {
	Code int
}

type Handler interface {
	Handle(req *Request) error
	Resize(tty *Resize) error
	Wait() (Response, error)
	Close() error
}

type HandlerFunc func(payload string) (Handler, error)

func (req *Tty) Resize() *Resize {
	return &Resize{
		Width:  req.Width,
		Height: req.Height,
	}
}
