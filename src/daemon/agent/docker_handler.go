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
}

func NewDockerClient() (*docker.Client, error) {
	return docker.NewClientFromEnv()
}

func NewDockerHandler(client *docker.Client, parser payload.Parser) (*DockerHandler, error) {
	handler := &DockerHandler{
		cli:    client,
		parser: parser,
	}
	return handler, nil
}

func (h *DockerHandler) Handle(req *HandleRequest) error {
	filter, err := h.parser.Parse(req.Payload)
	if err != nil {
		return err
	}

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

		if h.isMatched(filter, inspect) {
			matched = inspect
			break
		}
	}

	if matched == nil {
		return errors.New(fmt.Sprintf("Could not found container for %+v", filter))
	}

	log.Debugf("Found container %s", matched.ID[:10])

	return h.execCommand(matched, req)
}

func (h *DockerHandler) IsStarted() bool {
	return h.exec != nil
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

	log.Debugf("Container exec successfuly created %s", exec.ID[:10])

	success := make(chan struct{}, 1)
	started := make(chan error, 1)

	go func() {
		select {
		case <-success:
			success <- struct{}{}
			started <- nil
		case <-time.After(3 * time.Second):
			started <- errors.New("Could not wait exec within 3 seconds")
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

	log.Debugf("Container exec successfuly started %s", exec.ID[:10])

	if req.Tty != nil {
		if err := h.Resize(req.Tty.Resize()); err != nil {
			return err
		}
	}

	return nil
}

func (h *DockerHandler) Wait() error {
	if h.closer != nil {
		log.Debug("Starting wait for container exec response")
		if err := h.closer.Wait(); err != nil {
			return errors.New(fmt.Sprintf("Could wait container exec (%s)", err))
		}
	}
	return nil
}

func (h *DockerHandler) isMatched(filter *payload.Request, container *docker.Container) bool {
	if filter.ContainerId != "" && strings.HasPrefix(container.ID, filter.ContainerId) {
		return true
	}

	if filter.ContainerEnv != "" {
		for _, env := range container.Config.Env {
			log.Infof("env (%s) [%s]", container.Name, env)
			if env == filter.ContainerEnv {
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
			return errors.New(fmt.Sprintf("Could not close container exec (%s)", err))
		}

		h.exec = nil
		log.Debugf("Container exec successfuly closed")
	}

	return nil
}
