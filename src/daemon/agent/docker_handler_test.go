package agent

import (
	"daemon/payload"
	"daemon/testutils"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/require"
	"path"
	"runtime"
	"testing"
	"testing/iotest"
)

func Test_DockerHandler_shouldSuccessfullyRunInteractiveSession(t *testing.T) {
	cli := NewTestDockerClient(t)

	container := NewTestDockerContainer(t, cli, "FOO=bar", map[string]string{
		"foo": "bar",
	})
	defer RemoveTestDockerContainer(t, cli, container)

	filter := &payload.Request{
		ContainerId: container.ID,
	}

	handler := NewTestDockerHandler(t, cli, filter)
	defer CloseTestDockerHandler(t, handler)

	tty := &TtyRequest{
		Term:   "xterm",
		Width:  120,
		Height: 40,
	}

	pipe := testutils.NewTestingPipe()

	handleReq := &HandleRequest{
		Tty:    tty,
		Stdin:  iotest.NewReadLogger("[r]: ", pipe.IoReader()),
		Stdout: iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
		Stderr: iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
	}

	require.NoError(t, handler.Handle(handleReq))
	require.NoError(t, handler.Resize(handleReq.Tty.Resize()))

	pipe.SendString("echo term is $TERM\n")
	pipe.SendString("echo uname is $(uname)\n")
	pipe.SendString("echo complete.\n")

	require.NoError(t, pipe.WaitStringReceived("complete."))
	require.NoError(t, handler.Close())

	_, err := handler.Wait()
	require.NoError(t, err)

	require.Contains(t, pipe.String(), "echo term is $TERM\r\n")
	require.Contains(t, pipe.String(), "term is xterm\r\n")

	require.Contains(t, pipe.String(), "echo uname is $(uname)\r\n")
	require.Contains(t, pipe.String(), "uname is Linux\r\n")
}

func Test_DockerHandler_shouldSuccessfullyRunNonInteractiveSession(t *testing.T) {
	cli := NewTestDockerClient(t)

	container := NewTestDockerContainer(t, cli, "FOO=bar", map[string]string{})
	defer RemoveTestDockerContainer(t, cli, container)

	filter := &payload.Request{
		ContainerId: container.ID,
	}

	handler := NewTestDockerHandler(t, cli, filter)
	defer CloseTestDockerHandler(t, handler)

	pipe := testutils.NewTestingPipe()

	handleReq := &HandleRequest{
		Stdin:  iotest.NewReadLogger("[r]: ", pipe.IoReader()),
		Stdout: iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
		Stderr: iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
		Exec:   "ls -la ; echo complete.",
	}

	require.NoError(t, handler.Handle(handleReq))
	require.NoError(t, pipe.WaitStringReceived("complete."))
	require.NoError(t, handler.Close())

	_, err := handler.Wait()
	require.NoError(t, err)

	require.Contains(t, pipe.String(), ".dockerenv\n")
}

func Test_DockerHandler_shouldSuccessfullyFindContainers(t *testing.T) {
	cli := NewTestDockerClient(t)
	container := NewTestDockerContainer(t, cli, "ENV_NAME=envValue", map[string]string{
		"labelName": "labelValue",
	})
	defer RemoveTestDockerContainer(t, cli, container)

	simpleHandler := func(t *testing.T, filter *payload.Request) {
		handler := NewTestDockerHandler(t, cli, filter)
		defer CloseTestDockerHandler(t, handler)

		pipe := testutils.NewTestingPipe()

		handleReq := &HandleRequest{
			Stdin:  iotest.NewReadLogger("[r]: ", pipe.IoReader()),
			Stdout: iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
			Stderr: iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
			Exec:   "echo complete.",
		}

		require.NoError(t, handler.Handle(handleReq))
		require.NoError(t, pipe.WaitStringReceived("complete."))
		require.NoError(t, handler.Close())

		_, err := handler.Wait()
		require.NoError(t, err)
	}

	t.Run("container.ID", func(t *testing.T) {
		simpleHandler(t, &payload.Request{
			ContainerId: container.ID,
		})
	})

	t.Run("container.Env", func(t *testing.T) {
		simpleHandler(t, &payload.Request{
			ContainerEnv: "ENV_NAME=envValue",
		})
	})

	t.Run("container.Label", func(t *testing.T) {
		simpleHandler(t, &payload.Request{
			ContainerLabel: "labelName=labelValue",
		})
	})
}

func Test_DockerHandler_shouldFailToHandleRequests(t *testing.T) {
	cli := NewTestDockerClient(t)
	container := NewTestDockerContainer(t, cli, "FOO=BAR", map[string]string{})
	defer RemoveTestDockerContainer(t, cli, container)

	simpleHandler := func(t *testing.T, filter *payload.Request, expect string) {
		handler := NewTestDockerHandler(t, cli, filter)
		defer CloseTestDockerHandler(t, handler)

		pipe := testutils.NewTestingPipe()

		handleReq := &HandleRequest{
			Stdin:  iotest.NewReadLogger("[r]: ", pipe.IoReader()),
			Stdout: iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
			Stderr: iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
			Exec:   "true",
		}

		err := handler.Handle(handleReq)
		require.Error(t, err)
		require.Contains(t, err.Error(), expect)
		require.NoError(t, handler.Close())

		code, err := handler.Wait()
		require.Error(t, err)
		require.Equal(t, 1, code)
	}

	t.Run("container not found", func(t *testing.T) {
		simpleHandler(t, &payload.Request{ContainerId: "notFound"}, "Could not found container for ")
	})
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
