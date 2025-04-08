package main

import (
	"os/exec"
	"time"

	"github.com/charmbracelet/log"
)

type Process struct {
	ID      string
	Setup   func() *exec.Cmd
	Command *exec.Cmd
}

type Supervisor struct {
	Processes   []Process
	RestartTime time.Duration
	Running     bool
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		RestartTime: time.Second * 5,
	}
}

func (mgr *Supervisor) Add(id string, setup func() *exec.Cmd) {
	mgr.Processes = append(mgr.Processes, Process{
		ID:    id,
		Setup: setup,
	})
}

func (mgr *Supervisor) Run() {
	if mgr.Running {
		return
	}

	mgr.Running = true

	for _, process := range mgr.Processes {
		var run func()

		run = func() {
			process.Command = process.Setup()

			log.Infof("starting %s...", process.ID)

			err := process.Command.Run()
			if err != nil {
				log.Errorf("process %s: %s", process.ID, err)
			}

			time.Sleep(mgr.RestartTime)

			run()
		}

		go run()
	}

	select {}
}
