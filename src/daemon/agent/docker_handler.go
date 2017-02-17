package agent

import (
	"daemon/payload"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
	"strings"
	"time"
)

type DockerHandler struct {
	cli       *docker.Client
	parser    payload.Parser
	container *docker.Container
	exec      *docker.Exec
	closer    docker.CloseWaiter
	filter    *payload.Request
}

func NewDockerClient() (*docker.Client, error) {
	return docker.NewClientFromEnv()
}

func NewDockerHandler(client *docker.Client, filter *payload.Request) (*DockerHandler, error) {
	handler := &DockerHandler{
		cli:    client,
		filter: filter,
	}
	return handler, nil
}

func (h *DockerHandler) Handle(req *HandleRequest) error {
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
		return errors.New(fmt.Sprintf("Could not found container for %+v", h.filter))
	}

	log.Debugf("Found container %s", matched.ID[:10])

	return h.execCommand(matched, req)
}

func (h *DockerHandler) execCommand(container *docker.Container, req *HandleRequest) error {

	h.container = container

	createExecOptions := docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"/bin/sh"},
		Container:    container.ID,
	}

	if req.Tty != nil {
		createExecOptions.Cmd = []string{"/usr/bin/env", fmt.Sprintf("TERM=%s", req.Tty.Term), "sh"}
	}

	exec, err := h.cli.CreateExec(createExecOptions)
	if err != nil {
		return err
	}
	h.exec = exec

	log.Debugf("Container session successfuly created %s", exec.ID[:10])

	success := make(chan struct{}, 1)
	started := make(chan error, 1)

	go func() {
		select {
		case <-success:
			success <- struct{}{}
			started <- nil
		case <-time.After(3 * time.Second):
			started <- errors.New("Could not wait session within 3 seconds")
		}
	}()

	startExecOptions := docker.StartExecOptions{
		InputStream:  req.Reader,
		OutputStream: req.Writer,
		ErrorStream:  req.Writer,
		Detach:       false,
		Tty:          false,
		RawTerminal:  true,
		Success:      success,
	}

	if req.Tty != nil {
		startExecOptions.Tty = true
	}

	closer, err := h.cli.StartExecNonBlocking(exec.ID, startExecOptions)
	if err != nil {
		return err
	}
	h.closer = closer

	select {
	case err := <-started:
		if err != nil {
			return err
		}
	}

	log.Infof("Container session successfuly started %s", exec.ID[:10])

	if req.Tty != nil {
		if err := h.Resize(req.Tty.Resize()); err != nil {
			return err
		}
	}

	return nil
}

func (h *DockerHandler) Wait() error {
	if h.closer != nil {
		log.Debug("Starting wait for container session response")
		if err := h.closer.Wait(); err != nil {
			return errors.New(fmt.Sprintf("Could wait container session (%s)", err))
		}
	}
	return nil
}

func (h *DockerHandler) isMatched(container *docker.Container) bool {
	if h.filter.ContainerId != "" && strings.HasPrefix(container.ID, h.filter.ContainerId) {
		return true
	}

	if h.filter.ContainerEnv != "" {
		for _, env := range container.Config.Env {
			if env == h.filter.ContainerEnv {
				return true
			}
		}
	}

	return false
}

func (h *DockerHandler) Resize(req *ResizeRequest) error {
	if req != nil && h.exec != nil {
		err := h.cli.ResizeExecTTY(h.exec.ID, int(req.Height), int(req.Width))
		if err != nil {
			return errors.New(fmt.Sprintf("Could not resize tty (%s)", err))
		}
		log.Debugf("Tty successfuly resized %v", *req)
	}
	return nil
}

func (h *DockerHandler) Close() error {

	if h.exec != nil && h.closer != nil {
		err := h.closer.Close()
		if err != nil {
			return errors.New(fmt.Sprintf("Could not close container session (%s)", err))
		}

		h.exec = nil
		log.Info("Container session successfuly closed")
	}

	return nil
}
