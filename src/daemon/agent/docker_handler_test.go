package agent

import (
	"bytes"
	"daemon/payload"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/require"
	"path"
	"runtime"
	"testing"
	"testing/iotest"
	"time"
)

func Test_DockerHandler_shouldSuccessfullyAttachToContainerByEnv(t *testing.T) {
	cli := NewTestDockerClient(t)

	container := NewTestDockerContainer(t, cli, "FOO=bar", map[string]string{
		"foo": "bar",
	})
	defer RemoveTestDockerContainer(t, cli, container)

	filter := &payload.Request{
		ContainerEnv: "FOO=bar",
	}

	handler := NewTestDockerHandler(t, cli, filter)
	defer CloseTestDockerHandler(t, handler)

	tty := &TtyRequest{
		Term:   "xterm",
		Width:  120,
		Height: 40,
	}

	var writeBytes bytes.Buffer
	var readBytes bytes.Buffer

	writeBytes.WriteString("\n\n\n")

	handleReq := &HandleRequest{
		Tty:     tty,
		Reader:  iotest.NewReadLogger("[r]: ", &writeBytes),
		Writer:  iotest.NewWriteLogger("[w]: ", &readBytes),
	}

	require.NoError(t, handler.Handle(handleReq))
	require.NoError(t, handler.Resize(handleReq.Tty.Resize()))

	time.Sleep(3 * time.Second)

	require.NoError(t, handler.Close())
	require.NotEmpty(t, readBytes.String())
	require.Equal(t, "/ # \x1b[6n", readBytes.String())

	require.NoError(t, handler.Wait())
}

func CloseTestDockerHandler(t *testing.T, handler *DockerHandler) {
	if err := handler.Close(); err != nil {
		t.Error("Could not close docker handler")
	} else {
		t.Log("Docker handler successfully closed")
	}
}

func RemoveTestDockerContainer(t *testing.T, cli *docker.Client, container *docker.Container) {
	opts := docker.RemoveContainerOptions{
		ID:            container.ID,
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.RemoveContainer(opts); err != nil {
		t.Errorf("Could not remove container %s", container.ID)
	} else {
		t.Logf("Container %s successfully removed", container.ID)
	}
}

func NewTestDockerContainer(t *testing.T, cli *docker.Client, env string, labels map[string]string) *docker.Container {
	_, file, line, _ := runtime.Caller(1)

	name := path.Base(fmt.Sprintf("%s.%d", file, line))

	createOptions := docker.CreateContainerOptions{
		Name: name,
		Config: &docker.Config{
			Image:  "alpine",
			Cmd:    []string{"/bin/sh", "-c", "yes > /dev/null"},
			Env:    []string{env},
			Labels: labels,
		},
	}
	container, err := cli.CreateContainer(createOptions)
	require.NoError(t, err)
	require.NotNil(t, container)

	err = cli.StartContainer(container.ID, &docker.HostConfig{})
	if err != nil {
		RemoveTestDockerContainer(t, cli, container)
	}
	require.NoError(t, err)

	return container
}

func NewTestDockerClient(t *testing.T) *docker.Client {
	cli, err := NewDockerClient()
	require.NoError(t, err)
	require.NotNil(t, cli)

	return cli
}

func NewTestDockerHandler(t *testing.T, cli *docker.Client, filter *payload.Request) *DockerHandler {
	handler, err := NewDockerHandler(cli, filter)
	require.NoError(t, err)
	require.NotNil(t, handler)

	return handler
}