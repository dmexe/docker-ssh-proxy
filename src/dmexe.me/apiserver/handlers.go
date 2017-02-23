package apiserver

import (
	"github.com/Sirupsen/logrus"
	"net/http"
	"encoding/json"
	"github.com/gorilla/mux"
	"dmexe.me/utils"
	"fmt"
	"time"
)

type handlers struct {
	log *logrus.Entry
	provider Provider
	broker *utils.BytesBroker
}

func (h *handlers) getRouter() *mux.Router {
	root := mux.NewRouter()
	root.Methods("GET").Path("/health").HandlerFunc(h.handleHealth)

	api := root.PathPrefix("/a/v1").Subrouter()
	api.Methods("GET").Path("/tasks").HandlerFunc(h.handleTasks)
	api.Methods("GET").Path("/stream").HandlerFunc(h.handleStream)

	root.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		if t, err := route.GetPathTemplate(); err == nil {
			h.log.Debugf("Add handler %s", t)
		}
		return nil
	})

	return root
}

func (h *handlers) renderJSON(w http.ResponseWriter, code int, obj interface{}) {
	bb, err := json.Marshal(obj)
	if err != nil {
		h.log.Errorf("Could not marashal json (%s)", err)
		return
	}

	w.WriteHeader(code)
	w.Header().Add("Content-Type", "application/json")

	if _, err := w.Write(bb); err != nil {
		h.log.Errorf("Could not write response (%s)", err)
		return
	}
}

func (h *handlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.renderJSON(w, http.StatusOK, map[string]string{
		"status": "UP",
	})
}

func (h *handlers) handleTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.provider.GetTasks(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		h.renderJSON(w, http.StatusOK, tasks)
	}
}

func (h *handlers) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	messages := make(chan []byte)
	defer h.broker.Remove(messages)
	h.broker.Add(messages)

	notify := w.(http.CloseNotifier).CloseNotify()

	for {
		select {
		case <-notify:
			h.log.Debugf("Client closed stream")
			return

		case <- time.After(3 * time.Second):
			if _, err := fmt.Fprint(w, "data: n\n\n"); err != nil {
				h.log.Warnf("Could not write (%s)", err)
				return
			}
			flusher.Flush()

		case payload := <- messages:
			if _, err := fmt.Fprintf(w, "event: stream\ndata: %s\n\n", payload); err != nil {
				h.log.Warnf("Could not write (%s)", err)
				return
			}
			flusher.Flush()
		}
	}
}
