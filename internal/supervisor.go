package internal

import (
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/log"
)

type Process struct {
	ID    string
	Start func()
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

func (mgr *Supervisor) Add(id string, start func()) {
	mgr.Processes = append(mgr.Processes, Process{
		ID:    id,
		Start: start,
	})
}

// TODO: allow adding env as well

func (mgr *Supervisor) AddSimple(id string, command string, arg ...string) {
	mgr.Add(id, func() {
		cmd := exec.Command(command, arg...)

		if SUPERVISOR_LOGS {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
		}

		err := cmd.Run()

		if err != nil {
			log.Error(id, "err", err.Error())
		}
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
			log.Infof("starting %s...", process.ID)
			process.Start()
			time.Sleep(mgr.RestartTime)
			run()
		}

		go run()
	}

	select {}
}
