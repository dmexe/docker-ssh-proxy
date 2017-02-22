package apiserver

import (
	"context"
	"dmexe.me/utils"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"sync"
)

// ServerOptions contains parameters for a new Server instance
type ServerOptions struct {
	Host     string
	Port     uint
	Provider Provider
}

// Server instance
type Server struct {
	listenAddress string
	log           *logrus.Entry
	ctx           context.Context
	provider      Provider
	httpServer    *http.Server
}

// NewServer creates a new server using given context and options
func NewServer(ctx context.Context, opts ServerOptions) (*Server, error) {
	server := &Server{
		provider:      opts.Provider,
		listenAddress: fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		log:           utils.NewLogEntry("api.server"),
		ctx:           ctx,
	}
	return server, nil
}

func (s *Server) renderJSON(w http.ResponseWriter, code int, obj interface{}) {
	bb, err := json.Marshal(obj)
	if err != nil {
		s.log.Errorf("Could not marashal json (%s)", err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)

	if _, err := w.Write(bb); err != nil {
		s.log.Errorf("Could not write response (%s)", err)
		return
	}
}

func (s *Server) renderError(w http.ResponseWriter, err error) {
	s.renderJSON(w, http.StatusInternalServerError, map[string]string{
		"error": err.Error(),
	})
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	s.renderJSON(w, http.StatusOK, map[string]string{
		"status": "UP",
	})
}

func (s *Server) getTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.provider.GetTasks(s.ctx)
	if err != nil {
		s.renderError(w, err)
	} else {
		s.renderJSON(w, http.StatusOK, tasks)
	}
}

// Run server
func (s *Server) Run(wg *sync.WaitGroup) error {
	rootRouter := mux.NewRouter()
	rootRouter.Methods("GET").Path("/health").HandlerFunc(s.getHealth)

	apiRouter := rootRouter.PathPrefix("/a/v1").Subrouter()
	apiRouter.Methods("GET").Path("/tasks").HandlerFunc(s.getTasksHandler)

	rootRouter.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		if t, err := route.GetPathTemplate(); err != nil {
			s.log.Debugf("Route %s", t)
		}
		return nil
	})

	server := &http.Server{
		Handler: rootRouter,
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

		if err := s.httpServer.Shutdown(s.ctx); err != nil {
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
