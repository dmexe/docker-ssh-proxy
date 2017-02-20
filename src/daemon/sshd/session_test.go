package sshd

import (
	"daemon/handlers"
	"daemon/utils"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"testing"
	"testing/iotest"
	"time"
)

func Test_Session(t *testing.T) {

	t.Run("should successfuly", func(t *testing.T) {
		server := newTestServer(t, newEchoHandler(handlers.EchoHandlerErrors{}))
		defer closeTestServer(t, server)

		t.Run("run interactive session", func(t *testing.T) {
			session, closer := newTestSession(t, server.Addr(), "username")
			defer closer.Close()

			require.NoError(t, requestTty(session))

			pipe := setupSessionPipe(t, session)

			require.NoError(t, session.Shell())

			pipe.SendString("complete.\n")
			require.NoError(t, pipe.WaitString("complete."))

			require.NoError(t, requestResize(session))
		})

		t.Run("run non interactive session", func(t *testing.T) {
			session, closer := newTestSession(t, server.Addr(), "username")
			defer closer.Close()

			pipe := setupSessionPipe(t, session)

			require.NoError(t, session.Start("echo complete."))
			require.NoError(t, pipe.WaitString("complete."))
		})
	})

	testErr := errors.New("boom")

	t.Run("fail to create shell", func(t *testing.T) {
		failHandler := func(_ string) (handlers.Handler, error) {
			return nil, testErr
		}

		server := newTestServer(t, failHandler)
		defer closeTestServer(t, server)

		session, closer := newTestSession(t, server.Addr(), "username")
		defer closer.Close()

		require.NoError(t, requestTty(session))
		require.Error(t, session.Shell())
	})

	t.Run("fail to handle request", func(t *testing.T) {
		server := newTestServer(t, newEchoHandler(handlers.EchoHandlerErrors{
			Handle: testErr,
		}))
		defer closeTestServer(t, server)

		session, closer := newTestSession(t, server.Addr(), "username")
		defer closer.Close()

		require.NoError(t, requestTty(session))
		require.Error(t, session.Shell())
	})

	t.Run("fail to wait when handle completed", func(t *testing.T) {
		server := newTestServer(t, newEchoHandler(handlers.EchoHandlerErrors{
			Wait: testErr,
		}))
		defer closeTestServer(t, server)

		session, closer := newTestSession(t, server.Addr(), "username")
		defer closer.Close()

		pipe, err := session.StdinPipe()
		require.NoError(t, err)

		require.NoError(t, requestTty(session))
		require.NoError(t, session.Shell())

		time.Sleep(1 * time.Second)

		_, err = pipe.Write([]byte("test"))
		require.Error(t, err)
		require.Equal(t, "EOF", err.Error())
	})
}

func newEchoHandler(errors handlers.EchoHandlerErrors) handlers.HandlerFunc {
	return func(_ string) (handlers.Handler, error) {
		return handlers.NewEchoHandler(errors), nil
	}
}

func setupSessionPipe(t *testing.T, s *ssh.Session) *utils.BytesBackedPipe {
	pipe := utils.NewBytesBackedPipe()

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

func requestResize(s *ssh.Session) error {
	bb := make([]byte, 8)
	binary.BigEndian.PutUint32(bb, 120)
	binary.BigEndian.PutUint32(bb, 80)
	reply, err := s.SendRequest("window-change", true, bb)
	if !reply {
		return fmt.Errorf("window-change request doesn't completed expected response=true, actual=%t", reply)
	}
	return err
}

func requestTty(s *ssh.Session) error {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	return s.RequestPty("xterm", 80, 40, modes)
}

func newTestSession(t *testing.T, addr net.Addr, user string) (*ssh.Session, io.Closer) {
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

func closeTestServer(t *testing.T, server *Server) {
	if err := server.Close(); err != nil {
		t.Error(err)
	}

	if err := server.Wait(); err != nil {
		t.Error(err)
	}
}

func newTestServer(t *testing.T, handler handlers.HandlerFunc) *Server {
	opts := CreateServerOptions{
		ListenAddr:      "localhost:0",
		PrivateKeyBytes: newRsaPrivateKey(),
	}

	server, err := NewServer(opts, handler)
	require.NoError(t, err)
	require.NotNil(t, server)
	require.NoError(t, server.Run())

	return server
}

// newRsaPrivateKey creates RSA key for testing
func newRsaPrivateKey() []byte {
	key := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAvAO2zpHuvFuOGEpjwxdhDfT1JBQsuK1/LmENBgLZmCQsGt4Y
tRJSpzWsfIMZSiu4Fhwcc//6Fvt9SPpSOJVNaqVeIkXYGcaFXxZ/3msskYOokg+k
VKdGGC+qHab78FSNk3beEbw9eNl+b8PI/+DxYu7KJ7M+K3mNDpwXH4PGKO9LqBZ+
4RFvzYOulyDfTi1qaTAivch79fzWaFs7QKW0DrshZRkwlyIRJS3/d35GlJj2nVeR
FYiSex/kap+rn6/GWvn0RW+rJrUaeO2RxUHMkA3YXy4XBO92ICaRYG1jGWewceId
dIXeoiOBI795O/1x83DKA72+1TDaTyS/bnajZwIDAQABAoIBAQCK91PPKx4CKsnE
OneyYz1hS4VFvYOwnMw8Q4+UudaLFXFkCnTIoVpmLM3o2h1/LQFLlkuRkcoP4qKf
piXPnMsz4DbLrkQkCQ/bUI4Cn8S5aU7XZqhXyauNhO2ALURaRqS+MkXBZhkpkdha
U6PlPSYtscHJxjpVd+pCuix9JrD4319ZGRt80CmMoG440L2GJAbISe61dYMDcyO+
/wyCj+663hIAkzSOOULj62TaJdO31MyOoRv2E2R4wuURVTPd4VNIw79q6LRnm8tM
6EdCbRtUFQnwdsXOKoTw8sT62ZdUvaboBYKPnpwFtwcQeMQnYilL0rV5SCHySrMp
+vUuNv3BAoGBAPleij+JBCJvixqsEXIDjRRzDpdPsjyOiKRdfXqck2LquVUFeUyJ
pO+2jfrdr146rlPf5cEtIxi68WlC8VeVxwmNAl9dQCQDQcah2bywg9+7bE4XDRnX
wAM34KrtFIdIlS1joF/Yqn4Nc2c9rqyh0tFjnRwn9Mh3TPe7s1oV6Uy1AoGBAMED
h/7LdGMoZJX7uUVX+nLmW24Xb0j0QMnQKwUqtQAI1qXdSvee0EmnirSJhEEF3OoB
JVDHYTv1f7XxYfHZrvpk6J0ok3q07o5EI37xtjMtPv9+HZLRhcQxFEaFczqy55cG
EtboKh++fF5Z6Uk9f+aLXcxSGDf4m26fNNtMZV0rAoGAU9QLJ2aZBDZ5DaNQTgKR
l5FCE22QHjlQB+kBuIkQJs1/NeycJTWUQ50bx3xkaonRdpKqurDAvpyBcQA2/1lz
Smujo4lGeZS6tNpNxteTzU9FDk9DcS+M9cf/95WxM/UbaOG31OCSF8PPyqH6qT/R
DeCtvPxVllo8fn8TwLHi9o0CgYEAkQgzP0zn5r5qXpzoyWdjZMUdfKsVTw9iQ4Mt
YFOH8D+z8qxG8awfPMktG52diDJ8nkVAIeO/d4twbGm1vEJjDfmXJMhhkTm1a6dd
uLytuOTNyrOcSz8vMY3je145iKj4Bm5k56FKTdIXp9oNxp/0pGqij648Top7WPM+
h25vWEMCgYAWzHaHEWpJm0XgF1gJh1bEvfmtn2OYCdWZJb3K3x5duKn0c7wn20jT
q7GupcpqynvTvD8gOlQSwrh+xnqXmFf2h7FOGUG8PsNDEDlxVrymGei/yF5CFWaG
GrEm4Ai0fJiI4xK8/q4GWq60fnRKEHemC1CFiYyuIDegY5wLP6C88g==
-----END RSA PRIVATE KEY-----`

	return []byte(key)
}
