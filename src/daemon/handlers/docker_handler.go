package handlers

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
	container *docker.Container
	exec      *docker.Exec
	closer    docker.CloseWaiter
	filter    *payload.Request
	closed    bool
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
		return errors.New(fmt.Sprintf("Could not found container for %v", h.filter))
	}

	return h.execCommand(matched, req)
}

func (h *DockerHandler) execCommand(container *docker.Container, req *Request) error {

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
		args := []string{"/usr/bin/env"}
		if req.Tty != nil {
			args = append(args, fmt.Sprintf("TERM=%s", req.Tty.Term))
		}
		args = append(args, "sh", "-c", fmt.Sprintf("%s", req.Exec))
		log.Debugf("Container session args %v", args)
		createExecOptions.Cmd = args
	}

	session, err := h.cli.CreateExec(createExecOptions)
	if err != nil {
		return err
	}
	h.exec = session

	log.Debugf("Container session successfuly created %s", session.ID[:10])

	success := make(chan struct{}, 1)

	startExecOptions := docker.StartExecOptions{
		InputStream:  req.Stdin,
		OutputStream: req.Stdout,
		ErrorStream:  req.Stderr,
		Detach:       false,
		Tty:          false,
		RawTerminal:  true,
		Success:      success,
	}

	if req.Tty != nil {
		startExecOptions.Tty = true
	}

	closer, err := h.cli.StartExecNonBlocking(session.ID, startExecOptions)
	if err != nil {
		return err
	}
	h.closer = closer

	started := make(chan error, 1)

	go func() {
		select {
		case <-success:
			success <- struct{}{}
			started <- nil
		case <-time.After(15 * time.Second):
			started <- errors.New("Could not wait session within 15 seconds")
		}
	}()

	select {
	case err := <-started:
		if err != nil {
			return err
		}
	}

	log.Infof("Container session successfuly started %s", session.ID[:10])

	if req.Tty != nil {
		if err := h.Resize(req.Tty.Resize()); err != nil {
			return err
		}
	}

	return nil
}

func (h *DockerHandler) Wait() (int, error) {
	if h.closer != nil {
		log.Debug("Starting wait for container session response")
		if err := h.closer.Wait(); err != nil {
			return 1, errors.New(fmt.Sprintf("Could wait container session (%s)", err))
		}
	}

	return h.exitCode()
}

func (h *DockerHandler) exitCode() (int, error) {
	if h.exec == nil {
		return 1, errors.New("Exec instance is undefined")
	}

	inspect, err := h.cli.InspectExec(h.exec.ID)
	if err != nil {
		return 1, errors.New(fmt.Sprintf("Could not inspect exec=%s (%s)", h.exec.ID, err))
	}

	log.Debugf("Process exited with code %d", inspect.ExitCode)

	return inspect.ExitCode, nil
}

func (h *DockerHandler) isMatched(container *docker.Container) bool {
	if len(h.filter.ContainerId) > 8 && strings.HasPrefix(container.ID, h.filter.ContainerId) {
		log.Debugf("Match container by id=%s", container.ID)
		return true
	}

	if h.filter.ContainerEnv != "" {
		for _, env := range container.Config.Env {
			if env == h.filter.ContainerEnv {
				log.Debugf("Match container by env %s id=%s", env, container.ID)
				return true
			}
		}
	}

	if h.filter.ContainerLabel != "" {
		fields := strings.FieldsFunc(h.filter.ContainerLabel, func(r rune) bool {
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

	if h.closed {
		log.Warnf("Close session called multiple times")
		return nil
	}
	h.closed = true

	if h.exec != nil && h.closer != nil {
		err := h.closer.Close()
		if err != nil {
			return errors.New(fmt.Sprintf("Could not close container session (%s)", err))
		}
		log.Info("Container session successfuly closed")
	}

	return nil
}