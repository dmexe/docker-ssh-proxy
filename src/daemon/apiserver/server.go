package apiserver

import (
	"context"
	"daemon/utils"
	"fmt"
	"github.com/Sirupsen/logrus"
	//"github.com/gorilla/mux"
	"sync"
)

type ServerOptions struct {
	Host string
	Port uint
	Provider Provider
}

type Server struct {
	listenAddress string
	log           *logrus.Entry
	ctx           context.Context
	provider      Provider
}

func NewServer(ctx context.Context, opts ServerOptions) (*Server, error) {
	server := &Server{
		listenAddress: fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		log:           utils.NewLogEntry("ssh.server"),
		ctx:           ctx,
	}
	return server, nil
}

func (s *Server) Run(wg *sync.WaitGroup) error {
	//rootRouter := mux.NewRouter()
	//apiRouter := rootRouter.PathPrefix("a/v1")
	return nil
}
