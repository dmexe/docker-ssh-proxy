package handlers

import (
	"context"
	"dmexe.me/utils"
	"github.com/Sirupsen/logrus"
	"io"
)

// EchoHandlerErrors keeps all error request types
type EchoHandlerErrors struct {
	Handle error
	Wait   error
	Close  error
}

// EchoHandler used only in tests
type EchoHandler struct {
	completed chan error
	errors    EchoHandlerErrors
	log       *logrus.Entry
}

// NewEchoHandler creates a new handler
func NewEchoHandler(errors EchoHandlerErrors) *EchoHandler {
	return &EchoHandler{
		completed: make(chan error),
		errors:    errors,
		log:       utils.NewLogEntry("handler.echo"),
	}
}

// Handle request, just copy stdin to stdout
func (h *EchoHandler) Handle(_ context.Context, req *Request) (Response, error) {
	if h.errors.Handle != nil {
		return errResponse, h.errors.Handle
	}

	go func() {
		_, err := io.Copy(req.Stdout, req.Stdin)
		if err != nil {
			h.log.Warnf("Could not copy io streams (%s)", err)
		}
		h.completed <- err
	}()

	if req.Exec != "" {
		if _, err := req.Stdout.Write([]byte(req.Exec)); err != nil {
			return errResponse, err
		}
	}

	err := <-h.completed
	if err != nil {
		return errResponse, err
	}

	return Response{Code: 0}, nil
}

// Resize nothing
func (h *EchoHandler) Resize(tty *Resize) error {
	return nil
}

// Close nothing
func (h *EchoHandler) Close() error {
	if h.errors.Close != nil {
		return h.errors.Close
	}
	return nil
}
