package main

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	docker "github.com/fsouza/go-dockerclient"
	"strings"
	"time"
)

type DockerAgent struct {
	cli       *docker.Client
	jwt       *JwtPayloadParser
	container *docker.Container
	exec      *docker.Exec
	closer    docker.CloseWaiter
}

func NewDockerClient() (*docker.Client, error) {
	return docker.NewClientFromEnv()
}

func NewDockerAgent(client *docker.Client, jwt *JwtPayloadParser) (*DockerAgent, error) {
	agent := &DockerAgent{
		cli: client,
		jwt: jwt,
	}
	return agent, nil
}

func (agent *DockerAgent) Handle(req *AgentHandleRequest) error {
	payload, err := agent.jwt.Parse(req.Payload)
	if err != nil {
		return err
	}

	containers, err := agent.cli.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return err
	}

	var matched *docker.Container

	for _, container := range containers {
		inspect, err := agent.cli.InspectContainer(container.ID)
		if err != nil {
			return err
		}

		if agent.isMatched(payload, inspect) {
			matched = inspect
			break
		}
	}

	if matched == nil {
		return errors.New(fmt.Sprintf("Could not found container with %+v", payload))
	}

	log.Debugf("Found container %s", matched.ID[:10])

	return agent.execCommand(matched, req)
}

func (agent *DockerAgent) IsHandled() bool {
	return agent.exec != nil
}

func (agent *DockerAgent) execCommand(container *docker.Container, req *AgentHandleRequest) error {

	agent.container = container

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

	exec, err := agent.cli.CreateExec(createExecOptions)
	if err != nil {
		return err
	}
	agent.exec = exec

	log.Debugf("Container exec successfuly created %s", exec.ID[:10])

	success := make(chan struct{}, 1)
	started := make(chan error, 1)

	startExecOptions := docker.StartExecOptions{
		InputStream:  req.Reader,
		OutputStream: req.Writer,
		ErrorStream:  req.Writer,
		Detach:       false,
		Tty:          false,
		RawTerminal:  true,
		Success:      success,
	}

	go func() {
		select {
		case <-success:
			success <- struct{}{}
			started <- nil
		case <-time.After(3 * time.Second):
			started <- errors.New("Could not wait exec within 3 seconds")
		}
	}()

	if req.Tty != nil {
		startExecOptions.Tty = true
	}

	waiter, err := agent.cli.StartExecNonBlocking(exec.ID, startExecOptions)
	if err != nil {
		return err
	}
	agent.closer = waiter

	select {
	case err := <-started:
		if err != nil {
			return err
		}
	}

	log.Debugf("Container exec successfuly started %s", exec.ID[:10])

	if req.Tty != nil {
		if err := agent.Resize(req.Tty.Resize()); err != nil {
			return err
		}
	}

	return nil
}

func (agent *DockerAgent) Wait() error {
	if agent.closer != nil {
		log.Debug("Starting wait for container exec response")
		if err := agent.closer.Wait(); err != nil {
			return errors.New(fmt.Sprintf("Could wait container exec (%s)", err))
		}
	}
	return nil
}

func (agent *DockerAgent) isMatched(filter *Payload, container *docker.Container) bool {
	if filter.ContainerId != "" && strings.HasPrefix(container.ID, filter.ContainerId) {
		return true
	}

	if filter.ContainerEnv != "" {
		for _, env := range container.Config.Env {
			if env == filter.ContainerEnv {
				return true
			}
		}
	}

	return false
}

func (agent *DockerAgent) Resize(req *AgentResizeRequest) error {
	if req != nil && agent.exec != nil {
		err := agent.cli.ResizeExecTTY(agent.exec.ID, int(req.Height), int(req.Width))
		if err != nil {
			return errors.New(fmt.Sprintf("Could not resize tty (%s)", err))
		}
		log.Debugf("Tty successfuly resized %v", *req)
	}
	return nil
}

func (agent *DockerAgent) Close() error {

	if agent.exec != nil && agent.closer != nil {
		err := agent.closer.Close()
		if err != nil {
			return errors.New(fmt.Sprintf("Could not close container exec (%s)", err))
		}

		agent.exec = nil
		log.Debugf("Container exec successfuly closed")
	}

	return nil
}
