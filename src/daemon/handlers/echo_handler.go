package handlers

import (
	log "github.com/Sirupsen/logrus"
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
}

// NewEchoHandler creates a new handler
func NewEchoHandler(errors EchoHandlerErrors) *EchoHandler {
	return &EchoHandler{
		completed: make(chan error),
		errors:    errors,
	}
}

// Handle request, just copy stdin to stdout
func (h *EchoHandler) Handle(req *Request) error {
	go func() {
		_, err := io.Copy(req.Stdout, req.Stdin)
		if err != nil {
			log.Warnf("Could not copy io streams (%s)", err)
		}
		h.completed <- err
	}()

	if h.errors.Handle != nil {
		return h.errors.Handle
	}

	if req.Exec != "" {
		if _, err := req.Stdout.Write([]byte(req.Exec)); err != nil {
			return err
		}
	}

	return nil
}

// Resize nothing
func (h *EchoHandler) Resize(tty *Resize) error {
	return nil
}

// Wait until copy of io streams finished
func (h *EchoHandler) Wait() (Response, error) {
	if h.errors.Wait != nil {
		return Response{Code: 1}, h.errors.Wait
	}

	select {
	case err := <-h.completed:
		if err != nil {
			return Response{Code: 1}, err
		}
	}

	return Response{Code: 1}, nil
}

// Close nothing
func (h *EchoHandler) Close() error {
	if h.errors.Close != nil {
		return h.errors.Close
	}
	return nil
}
