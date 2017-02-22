package handlers

import (
	"context"
	"daemon/payloads"
	"daemon/utils"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path"
	"runtime"
	"testing"
	"testing/iotest"
	"time"
)

func Test_DockerHandler(t *testing.T) {
	cli := newTestDockerClient(t)

	container := newTestDockerContainer(t, cli, "ENV_NAME=envValue", map[string]string{
		"labelName": "labelValue",
	})
	defer removeTestDockerContainer(t, cli, container)

	t.Run("should run interactive session", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		payload := payloads.Payload{
			ContainerID: container.ID,
		}

		handler := newTestDockerHandler(t, cli)
		defer closeTestDockerHandler(t, handler)

		tty := &Tty{
			Term:   "xterm",
			Width:  120,
			Height: 40,
		}

		pipe := utils.NewBytesBackedPipe()

		handleReq := &Request{
			Tty:     tty,
			Stdin:   iotest.NewReadLogger("[r]: ", pipe.IoReader()),
			Stdout:  iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
			Stderr:  iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
			Payload: payload,
		}

		response := make(chan testResponse)

		go func() {
			resp, err := handler.Handle(ctx, handleReq)
			response <- testResponse{resp, err}
		}()

		// Wait until shell session started, otherwise sometimes got docker error 'containerd: process not found for container'
		time.Sleep(100 * time.Millisecond)
		require.NoError(t, handler.Resize(handleReq.Tty.Resize()))

		pipe.SendString("echo term is $TERM\n")
		pipe.SendString("echo uname is $(uname)\n")
		pipe.SendString("echo compl\\ete.\n")

		require.NoError(t, pipe.WaitString("complete."))
		require.NoError(t, handler.Close())

		require.Contains(t, pipe.String(), "echo term is $TERM\r\n")
		require.Contains(t, pipe.String(), "term is xterm\r\n")
		require.Contains(t, pipe.String(), "echo uname is $(uname)\r\n")
		require.Contains(t, pipe.String(), "uname is Linux\r\n")

		select {
		case resp := <-response:
			require.NoError(t, resp.err)
			require.Equal(t, 0, resp.Response.Code)
		case <-time.After(1 * time.Second):
			require.FailNow(t, "Could not wait response within 1s")
		}
	})

	t.Run("should run non interactive session", func(t *testing.T) {
		payload := payloads.Payload{
			ContainerID: container.ID,
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		handler := newTestDockerHandler(t, cli)
		defer closeTestDockerHandler(t, handler)

		pipe := utils.NewBytesBackedPipe()

		handleReq := &Request{
			Stdin:   iotest.NewReadLogger("[r]: ", pipe.IoReader()),
			Stdout:  iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
			Stderr:  iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
			Exec:    "sh -c \"ls -la ; echo compl\\ete.\"",
			Payload: payload,
		}

		response := make(chan testResponse)

		go func() {
			resp, err := handler.Handle(ctx, handleReq)
			response <- testResponse{resp, err}
		}()

		require.NoError(t, pipe.WaitString("complete."))
		require.NoError(t, handler.Close())
		require.Contains(t, pipe.String(), ".dockerenv\n")

		select {
		case resp := <-response:
			require.NoError(t, resp.err)
			require.Equal(t, 0, resp.Response.Code)
		case <-time.After(1 * time.Second):
			require.FailNow(t, "Could not wait response within 1s")
		}
	})

	t.Run("should find containers", func(t *testing.T) {

		simpleHandler := func(t *testing.T, payload payloads.Payload) {
			handler := newTestDockerHandler(t, cli)
			defer closeTestDockerHandler(t, handler)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pipe := utils.NewBytesBackedPipe()

			handleReq := &Request{
				Stdin:   iotest.NewReadLogger("[r]: ", pipe.IoReader()),
				Stdout:  iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
				Stderr:  iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
				Exec:    "echo compl\\ete.",
				Payload: payload,
			}

			response := make(chan testResponse)

			go func() {
				resp, err := handler.Handle(ctx, handleReq)
				response <- testResponse{resp, err}
			}()

			require.NoError(t, pipe.WaitString("complete."))

			select {
			case <-time.After(1 * time.Second):
				require.FailNow(t, "Could not wait response within 1s")
			case resp := <-response:
				require.NoError(t, resp.err)
				require.Equal(t, 0, resp.Response.Code)
			}
		}

		t.Run("container.ID", func(t *testing.T) {
			simpleHandler(t, payloads.Payload{
				ContainerID: container.ID,
			})
		})

		t.Run("container.Env", func(t *testing.T) {
			simpleHandler(t, payloads.Payload{
				ContainerEnv: "ENV_NAME=envValue",
			})
		})

		t.Run("container.Label", func(t *testing.T) {
			simpleHandler(t, payloads.Payload{
				ContainerLabel: "labelName=labelValue",
			})
		})
	})

	t.Run("fail when container not found", func(t *testing.T) {
		handler := newTestDockerHandler(t, cli)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pipe := utils.NewBytesBackedPipe()

		handleReq := &Request{
			Stdin:   iotest.NewReadLogger("[r]: ", pipe.IoReader()),
			Stdout:  iotest.NewWriteLogger("[w]: ", pipe.IoWriter()),
			Stderr:  iotest.NewWriteLogger("[e]: ", pipe.IoWriter()),
			Exec:    "true",
			Payload: payloads.Payload{ContainerID: "notFound"},
		}

		response := make(chan testResponse)

		go func() {
			resp, err := handler.Handle(ctx, handleReq)
			response <- testResponse{resp, err}
		}()

		require.NoError(t, handler.Close())

		select {
		case resp := <-response:
			assert.Error(t, resp.err)
			assert.Contains(t, resp.err.Error(), "Could not found container for")
			assert.Equal(t, 255, resp.Response.Code)
		case <-time.After(1 * time.Second):
			require.FailNow(t, "Could not wait response within 1s")
		}
	})
}

type testResponse struct {
	Response Response
	err      error
}

func closeTestDockerHandler(t *testing.T, handler *DockerHandler) {
	if err := handler.Close(); err != nil {
		t.Error("Could not close docker handler")
	}
}

func removeTestDockerContainer(t *testing.T, cli *docker.Client, container *docker.Container) {
	opts := docker.RemoveContainerOptions{
		ID:            container.ID,
		RemoveVolumes: true,
		Force:         true,
	}

	if err := cli.RemoveContainer(opts); err != nil {
		t.Errorf("Could not remove container %s", container.ID)
	}
}

func newTestDockerContainer(t *testing.T, cli *docker.Client, env string, labels map[string]string) *docker.Container {
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
	require.NoError(t, err, name)
	require.NotNil(t, container)

	err = cli.StartContainer(container.ID, &docker.HostConfig{})
	if err != nil {
		removeTestDockerContainer(t, cli, container)
	}
	require.NoError(t, err, name)

	t.Logf("Container successfully created (%s)", name)

	return container
}

func newTestDockerClient(t *testing.T) *docker.Client {
	cli, err := NewDockerClientFromEnv()

	require.NoError(t, err)
	require.NotNil(t, cli)

	return cli
}

func newTestDockerHandler(t *testing.T, cli *docker.Client) *DockerHandler {
	handler, err := NewDockerHandler(DockerHandlerOptions{
		Client: cli,
	})

	require.NoError(t, err)
	require.NotNil(t, handler)

	return handler
}
