package handlers

import (
	"daemon/payloads"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/shlex"
	"strings"
)

type DockerHandler struct {
	cli       *docker.Client
	container *docker.Container
	session   *docker.Exec
	closer    docker.CloseWaiter
	payload   *payloads.Payload
	closed    bool
}

func NewDockerClient() (*docker.Client, error) {
	return docker.NewClientFromEnv()
}

func NewDockerHandler(client *docker.Client, payload *payloads.Payload) (*DockerHandler, error) {
	handler := &DockerHandler{
		cli:     client,
		payload: payload,
	}
	return handler, nil
}

func (h *DockerHandler) Handle(req *Request) error {
	containers, err := h.cli.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	var matched *docker.Container

	for _, container := range containers {
		inspect, err := h.cli.InspectContainer(container.ID)
		if err != nil {
			return err
		}

		if h.isMatched(inspect) {
			matched = inspect
			break
		}
	}

	if matched == nil {
		return fmt.Errorf("Could not found container for %v", h.payload)
	}

	return h.startSession(matched, req)
}

func (h *DockerHandler) startSession(container *docker.Container, req *Request) error {

	h.container = container

	createExecOptions := docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		Cmd:          []string{"/bin/sh"},
		Container:    container.ID,
	}

	if req.Tty != nil {
		createExecOptions.Tty = true
	}

	if req.Tty != nil && req.Exec == "" {
		createExecOptions.Cmd = []string{"/usr/bin/env", fmt.Sprintf("TERM=%s", req.Tty.Term), "sh"}
	}

	if req.Exec != "" {
		cmdline := []string{"/usr/bin/env"}
		if req.Tty != nil {
			cmdline = append(cmdline, fmt.Sprintf("TERM=%s", req.Tty.Term))
		}

		args, err := shlex.Split(req.Exec)
		if err != nil {
			return err
		}

		cmdline = append(cmdline, args...)
		log.Debugf("Container session with cmdline %v", cmdline)
		createExecOptions.Cmd = cmdline
	}

	session, err := h.cli.CreateExec(createExecOptions)
	if err != nil {
		return err
	}
	h.session = session

	log.Debugf("Container session successfuly created %s", session.ID[:10])

	success := make(chan struct{})

	startExecOptions := docker.StartExecOptions{
		InputStream:  req.Stdin,
		OutputStream: req.Stdout,
		ErrorStream:  req.Stderr,
		Detach:       false,
		Tty:          false,
		RawTerminal:  false,
		Success:      success,
	}

	if req.Tty != nil {
		startExecOptions.RawTerminal = true
		startExecOptions.Tty = true
	}

	go func() {
		select {
		case <-success:
			success <- struct{}{}

			if req.Tty != nil {
				if err := h.Resize(req.Tty.Resize()); err != nil {
					log.Errorf("Could not resize tty (%s)", err)
				}
			}
		}
	}()

	closer, err := h.cli.StartExecNonBlocking(session.ID, startExecOptions)
	if err != nil {
		return err
	}
	h.closer = closer

	log.Infof("Container session successfuly started %s", session.ID[:10])

	return nil
}

func (h *DockerHandler) Wait() (Response, error) {
	if h.closer != nil {
		log.Debug("Starting wait for container session response")
		if err := h.closer.Wait(); err != nil {
			return Response{Code: 1}, fmt.Errorf("Could wait container session (%s)", err)
		}
	}

	if h.session == nil {
		return Response{Code: 1}, errors.New("Exec instance is undefined")
	}

	inspect, err := h.cli.InspectExec(h.session.ID)
	if err != nil {
		return Response{Code: 1}, fmt.Errorf("Could not inspect session=%s (%s)", h.session.ID, err)
	}

	log.Debugf("Process exited with code %d", inspect.ExitCode)

	return Response{Code: inspect.ExitCode}, nil
}

func (h *DockerHandler) isMatched(container *docker.Container) bool {
	if len(h.payload.ContainerID) > 8 && strings.HasPrefix(container.ID, h.payload.ContainerID) {
		log.Debugf("Match container by id=%s", container.ID)
		return true
	}

	if h.payload.ContainerEnv != "" {
		for _, env := range container.Config.Env {
			if env == h.payload.ContainerEnv {
				log.Debugf("Match container by env %s id=%s", env, container.ID)
				return true
			}
		}
	}

	if h.payload.ContainerLabel != "" {
		fields := strings.FieldsFunc(h.payload.ContainerLabel, func(r rune) bool {
			return r == '='
		})

		if len(fields) == 2 {
			fieldName := fields[0]
			fieldValue := fields[1]

			for name, value := range container.Config.Labels {
				if name == fieldName && value == fieldValue {
					log.Debugf("Match container by label %s=%s id=%s", name, value, container.ID)
					return true
				}
			}
		}
	}

	return false
}

func (h *DockerHandler) Resize(req *Resize) error {
	if req != nil && h.session != nil {
		err := h.cli.ResizeExecTTY(h.session.ID, int(req.Height), int(req.Width))
		if err != nil {
			return fmt.Errorf("Could not resize tty (%s)", err)
		}
		log.Debugf("Tty successfuly resized to %v", *req)
	}
	return nil
}

func (h *DockerHandler) Close() error {
	if h.closed {
		log.Warnf("Close session called multiple times")
		return nil
	}
	h.closed = true

	if h.session != nil && h.closer != nil {
		err := h.closer.Close()
		if err != nil {
			return fmt.Errorf("Could not close container session (%s)", err)
		}
		log.Info("Container session successfuly closed")
	}
	return nil
}
