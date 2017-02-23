package apiserver

import (
	"context"
	"dmexe.me/utils"
	"fmt"
	"github.com/Sirupsen/logrus"
	ghandlers "github.com/gorilla/handlers"
	"net"
	"net/http"
	"sync"
)

// ServerOptions contains parameters for a new Server instance
type ServerOptions struct {
	Host     string
	Port     uint
	Provider Provider
	Broker   *Broker
}

// Server instance
type Server struct {
	listenAddress string
	log           *logrus.Entry
	ctx           context.Context
	provider      Provider
	httpServer    *http.Server
	broker        *Broker
}

// NewServer creates a new server using given context and options
func NewServer(ctx context.Context, opts ServerOptions) (*Server, error) {
	server := &Server{
		provider:      opts.Provider,
		listenAddress: fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		log:           utils.NewLogEntry("api.server"),
		ctx:           ctx,
		broker:        opts.Broker,
	}
	return server, nil
}

// Run server
func (s *Server) Run(wg *sync.WaitGroup) error {

	h := &handlers{
		log:      s.log,
		provider: s.provider,
		broker:   s.broker,
	}

	router := h.getRouter()
	server := &http.Server{
		Handler: ghandlers.CORS()(router),
	}

	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return fmt.Errorf("Could not listen on %s (%s)", s.listenAddress, err)
	}
	s.httpServer = server
	s.log.Printf("Listening on %s...", s.listenAddress)

	go func() {
		<-s.ctx.Done()
		s.log.Debugf("Context done")
		if s.httpServer == nil {
			return
		}

		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			s.log.Errorf("Could not shutdown (%s)", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer s.log.Debugf("Stop accepting incoming connections")

		err := server.Serve(listener)
		if err != nil && err.Error() == "http: Server closed" {
			return
		}

		if err != nil {
			s.log.Errorf("Could not serve (%s)", err)
		}
	}()

	return nil
}
