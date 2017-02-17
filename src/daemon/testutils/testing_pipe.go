package testutils

import (
	"bytes"
	"strings"
	"time"
	"errors"
	"io"
	"sync"
)

type TestingPipe struct {
	sync.RWMutex
	ReadIn *io.PipeReader
	ReadOut *io.PipeWriter
	WriteIn *io.PipeReader
	WriteOut *io.PipeWriter
	Bytes bytes.Buffer
	readComplete chan error
}

func NewTestingPipe() *TestingPipe {
	readIn, readOut := io.Pipe()
	writeIn, writeOut := io.Pipe()

	pipe := &TestingPipe{
		ReadIn: readIn,
		ReadOut: readOut,
		WriteIn: writeIn,
		WriteOut: writeOut,
		readComplete: make(chan error),
	}

	pipe.readAsync()
	return pipe
}

func (p *TestingPipe) String() string {
	return p.Bytes.String()
}

func (p *TestingPipe) IoReader() io.Reader {
	return p.WriteIn
}

func (p *TestingPipe) IoWriter() io.Writer {
	return p.ReadOut
}

func (p *TestingPipe) readAsync() {
	go func() {
		buf := make([]byte, 128)
		for {
			sz, err := p.ReadIn.Read(buf)
			if err != nil && err.Error() == "EOF" {
				p.readComplete <- nil
				return
			}
			if err != nil {
				p.readComplete <- err
				return
			}

			p.Lock()
			p.Bytes.Write(buf[0:sz])
			p.Unlock()
		}
	}()
}

func (p *TestingPipe) WaitStringReceived(str string) error {
	finished := make(chan bool)
	defer close(finished)

	go func() {
		for {
			select {
			case <- time.After(100 * time.Millisecond):
				p.RLock()
				if strings.Contains(p.Bytes.String(), str) {
					p.RUnlock()
					finished <- true
					return
				}
				p.RUnlock()
			}
		}
	}()

	select {
	case err := <-p.readComplete:
		if err != nil {
			return err
		}
	case <-finished:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("Could wait response within 10 seconds")
	}

	return nil
}

func (p *TestingPipe) SendString(str string) error {
	c := make(chan error)
	defer close(c)
	go func() {
		_, err := p.WriteOut.Write([]byte(str))
		c <- err
	}()

	select {
	case err := <-c:
		return err
	case <- time.After(3 * time.Second):
		return errors.New("Could wait write response within 3 seconds")
	}
}
