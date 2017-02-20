package handlers

import (
	"io"
)

// Resize request
type Resize struct {
	Width  uint32
	Height uint32
}

// Tty type
type Tty struct {
	Term   string
	Width  uint32
	Height uint32
}

// Request for handler
type Request struct {
	Tty    *Tty
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Exec   string
}

// Response from handler
type Response struct {
	Code int
}

// Handler generic interface
type Handler interface {
	io.Closer
	Handle(req *Request) error
	Resize(tty *Resize) error
	Wait() (Response, error)
}

// HandlerFunc creates a new handler, it's just wrapper around Handler constructors, converts payload string to
// payloads.Payload  and calls underlayer handler
type HandlerFunc func(payload string) (Handler, error)

// Resize request from Tty
func (req *Tty) Resize() *Resize {
	return &Resize{
		Width:  req.Width,
		Height: req.Height,
	}
}
