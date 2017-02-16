package main

/*
import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	pty "github.com/kr/pty"
	"io"
	"os"
	"os/exec"
	"sync"
)

type BashIoAgent struct {
	file    *os.File
	command *exec.Cmd
}

func NewBashIoAgent() (*BashIoAgent, error) {
	return &BashIoAgent{}, nil
}

func (bash *BashIoAgent) Resize(req *IoAgentTtyRequest) error {
	if req != nil && bash.file != nil {
		SetWinsize(bash.file.Fd(), req.Width, req.Height)
		log.Debugf("Resize TTY %v", req)
	}
	return nil
}

func (bash *BashIoAgent) Handle(req *IoAgentHandleRequest) error {
	command := exec.Command("/usr/local/bin/docker", "exec", "-it", "postgres", "sh")
	bash.command = command

	fd, err := pty.Start(command)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not start pty (%s)", err))
	}

	bash.file = fd

	err = bash.Resize(req.TtyRequest)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var once sync.Once
	wg.Add(1)

	go func() {
		bytes, err := io.Copy(req.Writer, bash.file)
		if err != nil {
			log.Errorf("Failed to copy stream (%s)", err)
		} else {
			log.Debugf("Write complete, %agent bytes", bytes)
		}
		once.Do(wg.Done)
	}()

	go func() {
		bytes, err := io.Copy(bash.file, req.Reader)
		if err != nil {
			log.Errorf("Failed to copy stream (%s)", err)
		} else {
			log.Debugf("Read complete, %agent bytes", bytes)
		}
		once.Do(wg.Done)
	}()

	wg.Wait()

	return nil
}

func (bash *BashIoAgent) Close() error {
	if bash.command == nil {
		return nil
	}

	err := bash.command.Process.Kill()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to kill shell (%s)", err))
	}

	bash.command = nil
	log.Infof("Shell successfuly closed")

	return nil
}
*/
