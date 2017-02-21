package handlers

import (
	"context"
	"daemon/payloads"
	"io"
)

var (
	unhandledErrResponse = Response{Code: 255}
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
	Tty     *Tty
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Exec    string
	Payload payloads.Payload
}

// Response from handler
type Response struct {
	Code int
}

// HandlerFunc is a factory method
type HandlerFunc func() (Handler, error)

// Handler generic interface
type Handler interface {
	io.Closer
	Handle(ctx context.Context, req *Request) (Response, error)
	Resize(tty *Resize) error
}

// Resize request from Tty
func (req *Tty) Resize() *Resize {
	return &Resize{
		Width:  req.Width,
		Height: req.Height,
	}
}
