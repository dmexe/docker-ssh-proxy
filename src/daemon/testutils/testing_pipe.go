package testutils

import (
	"bytes"
	"strings"
	"time"
	"errors"
	"io"
)

type TestingPipe struct {
	ReadIn *io.PipeReader
	ReadOut *io.PipeWriter
	WriteIn *io.PipeReader
	WriteOut *io.PipeWriter
	ReadCompleted chan error
	Bytes bytes.Buffer
}

func NewTestingPipe() *TestingPipe {
	readIn, readOut := io.Pipe()
	writeIn, writeOut := io.Pipe()

	pipe := &TestingPipe{
		ReadIn: readIn,
		ReadOut: readOut,
		WriteIn: writeIn,
		WriteOut: writeOut,
		ReadCompleted: make(chan error),
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
		_, err := p.Bytes.ReadFrom(p.ReadIn)
		p.ReadCompleted <- err
	}()
}

func (p *TestingPipe) WaitStringReceived(str string) error {
	return WaitForStringAppearedInBuffer(str, &p.Bytes)
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

func WaitForStringAppearedInBuffer(str string, bb *bytes.Buffer) error {
	complete := make(chan bool)
	defer close(complete)

	go func() {
		for {
			if strings.Contains(bb.String(), str) {
				complete <- true
				break
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	select {
	case <-complete:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("Could wait response within 10 seconds")
	}
}
