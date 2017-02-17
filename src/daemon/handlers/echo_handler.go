package handlers

import (
	log "github.com/Sirupsen/logrus"
	"io"
)

type EchoHandlerErrors struct {
	Handle error
	Wait   error
	Close  error
}

type EchoHandler struct {
	completed chan error
	errors    EchoHandlerErrors
}

func NewEchoHandler(errors EchoHandlerErrors) *EchoHandler {
	return &EchoHandler{
		completed: make(chan error),
		errors:    errors,
	}
}

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

func (h *EchoHandler) Resize(tty *Resize) error {
	return nil
}

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

func (h *EchoHandler) Close() error {
	if h.errors.Close != nil {
		return h.errors.Close
	}
	return nil
}
