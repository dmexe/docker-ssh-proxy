package sshd

import (
	"daemon/agent"
	"daemon/testutils"
	"errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"testing"
	"testing/iotest"
	"time"
)

func Test_Session_shouldSuccessfullyHandleRequests(t *testing.T) {
	server := NewTestServer(t, NewEchoHandler(agent.EchoHandlerErrors{}))
	defer CloseTestServer(t, server)

	t.Run("interactive", func(t *testing.T) {
		session, closer := NewTestSession(t, server.Addr(), "username")
		defer closer.Close()

		require.NoError(t, RequestTty(session))

		pipe := SetupSessionPipe(t, session)

		require.NoError(t, session.Shell())
		pipe.SendString("complete.\n")
		require.NoError(t, pipe.WaitStringReceived("complete."))
	})

	t.Run("non interactive", func(t *testing.T) {
		session, closer := NewTestSession(t, server.Addr(), "username")
		defer closer.Close()

		pipe := SetupSessionPipe(t, session)

		require.NoError(t, session.Start("echo complete."))
		require.NoError(t, pipe.WaitStringReceived("complete."))
	})
}

func Test_Session_shouldFailToCreateRequest(t *testing.T) {
	testErr := errors.New("boom!")

	failHandler := func(_ string) (agent.Handler, error) {
		return nil, testErr
	}

	server := NewTestServer(t, failHandler)
	defer CloseTestServer(t, server)

	session, closer := NewTestSession(t, server.Addr(), "username")
	defer closer.Close()

	require.NoError(t, RequestTty(session))
	require.Error(t, session.Shell())
}

func Test_Session_shouldFailToHandleRequest(t *testing.T) {
	boom := errors.New("boom!")

	server := NewTestServer(t, NewEchoHandler(agent.EchoHandlerErrors{
		Handle: boom,
	}))
	defer CloseTestServer(t, server)

	session, closer := NewTestSession(t, server.Addr(), "username")
	defer closer.Close()

	require.NoError(t, RequestTty(session))
	require.Error(t, session.Shell())
}

func Test_Session_shouldFailToWaitAgent(t *testing.T) {
	boom := errors.New("boom!")

	server := NewTestServer(t, NewEchoHandler(agent.EchoHandlerErrors{
		Wait: boom,
	}))
	defer CloseTestServer(t, server)

	session, closer := NewTestSession(t, server.Addr(), "username")
	defer closer.Close()

	pipe, err := session.StdinPipe()
	require.NoError(t, err)

	require.NoError(t, RequestTty(session))
	require.NoError(t, session.Shell())

	time.Sleep(1 * time.Second)

	_, err = pipe.Write([]byte("test"))
	require.Error(t, err)
	require.Equal(t, "EOF", err.Error())
}

func NewEchoHandler(errors agent.EchoHandlerErrors) agent.CreateHandler {
	return func(_ string) (agent.Handler, error) {
		return agent.NewEchoHandler(errors), nil
	}
}

func SetupSessionPipe(t *testing.T, s *ssh.Session) *testutils.TestingPipe {
	pipe := testutils.NewTestingPipe()

	stdin, err := s.StdinPipe()
	require.NoError(t, err)
	require.NotNil(t, stdin)

	stdout, err := s.StdoutPipe()
	require.NoError(t, err)
	require.NotNil(t, stdout)

	stderr, err := s.StderrPipe()
	require.NoError(t, err)
	require.NotNil(t, stderr)

	go io.Copy(iotest.NewWriteLogger("[r]: ", stdin), pipe.IoReader())
	go io.Copy(pipe.IoWriter(), iotest.NewReadLogger("[w]: ", stdout))
	go io.Copy(pipe.IoWriter(), iotest.NewReadLogger("[e]: ", stderr))

	return pipe
}

func RequestTty(s *ssh.Session) error {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	return s.RequestPty("xterm", 80, 40, modes)
}

func NewTestSession(t *testing.T, addr net.Addr, user string) (*ssh.Session, io.Closer) {
	config := &ssh.ClientConfig{
		User: user,
	}

	sshConn, err := ssh.Dial("tcp", addr.String(), config)
	require.NoError(t, err)
	require.NotNil(t, sshConn)

	sshSession, err := sshConn.NewSession()
	if err != nil {
		sshConn.Close()
	}

	require.NoError(t, err)
	require.NotNil(t, sshSession)

	return sshSession, sshConn
}

func CloseTestServer(t *testing.T, server *Server) {
	if err := server.Close(); err != nil {
		t.Error(err)
	}

	if err := server.Wait(); err != nil {
		t.Error(err)
	}
}

func NewTestServer(t *testing.T, handler agent.CreateHandler) *Server {
	opts := CreateServerOptions{
		ListenAddr:      "localhost:0",
		PrivateKeyBytes: testutils.NewRsaPrivateKey(),
	}

	server, err := NewServer(opts, handler)
	require.NoError(t, err)
	require.NotNil(t, server)
	require.NoError(t, server.Start())

	return server
}
