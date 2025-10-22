package supervisor

import (
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/makinori/inu-desktop/src/config"
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

func New() *Supervisor {
	return &Supervisor{
		RestartTime: time.Second * 5,
	}
}

func (supervisor *Supervisor) Add(id string, start func()) {
	supervisor.Processes = append(supervisor.Processes, Process{
		ID:    id,
		Start: start,
	})
}

type Command struct {
	ID      string
	Command string
	Args    []string
	Env     []string
}

func (supervisor *Supervisor) AddCommand(command Command) {
	supervisor.Add(command.ID, func() {
		cmd := exec.Command(command.Command, command.Args...)
		cmd.Env = command.Env

		if config.SUPERVISOR_LOGS {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
		}

		err := cmd.Run()

		if err != nil {
			slog.Error(command.ID, "err", err.Error())
		}
	})
}

func (supervisor *Supervisor) Run() {
	if supervisor.Running {
		return
	}

	supervisor.Running = true

	for _, process := range supervisor.Processes {
		var run func()

		run = func() {
			slog.Info("starting " + process.ID + "...")
			process.Start()
			time.Sleep(supervisor.RestartTime)
			run()
		}

		go run()
	}

	select {}
}
