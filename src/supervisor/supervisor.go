package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/makinori/inu-desktop/src/config"
)

type Process struct {
	ID         string
	Start      func() error
	Stop       func()
	Running    bool
	nowRunning chan struct{}
}

type Supervisor struct {
	Processes   []*Process
	RestartTime time.Duration
	Running     bool
}

func New() *Supervisor {
	return &Supervisor{
		RestartTime: time.Second * 5,
	}
}

func (supervisor *Supervisor) AddSimple(id string, start func() error) {
	supervisor.Processes = append(supervisor.Processes, &Process{
		ID:         id,
		Start:      start,
		Running:    true,
		nowRunning: make(chan struct{}, 1),
	})
}

type Command struct {
	ID          string
	Command     string
	Args        []string
	Env         []string
	Dir         string
	NoAutoStart bool
}

func (supervisor *Supervisor) AddCommand(command Command) {
	var process *Process = &Process{
		ID:         command.ID,
		Running:    !command.NoAutoStart,
		nowRunning: make(chan struct{}, 1),
	}

	process.Start = func() error {
		ctx, stop := context.WithCancel(
			context.Background(),
		)

		process.Stop = func() {
			slog.Info("stopping " + process.ID + "...")
			stop()
		}

		cmd := exec.CommandContext(ctx, command.Command, command.Args...)
		cmd.Env = command.Env
		cmd.Dir = command.Dir

		if config.SUPERVISOR_LOGS {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stdout
		}

		err := cmd.Run()

		// dont print error if the context was stopped
		if ctx.Err() != nil {
			return nil
		}

		return err
	}

	supervisor.Processes = append(supervisor.Processes, process)
}

func (supervisor *Supervisor) processLoop(process *Process) {
	if process.Running {
		slog.Info("starting " + process.ID + "...")
		err := process.Start()
		if err != nil {
			slog.Error(process.ID, "err", err.Error())
			time.Sleep(supervisor.RestartTime)
		}
	} else {
		<-process.nowRunning
	}
	supervisor.processLoop(process)
}

func (supervisor *Supervisor) findByID(id string) *Process {
	for _, process := range supervisor.Processes {
		if process.ID == id {
			return process
		}
	}
	return nil
}

func (supervisor *Supervisor) Start(id string) error {
	process := supervisor.findByID(id)
	if process == nil {
		return errors.New("failed to find process")
	}

	if process.Running {
		return errors.New("process already running")
	}

	process.Running = true

	// avoid blocking
	if len(process.nowRunning) == 0 {
		process.nowRunning <- struct{}{}
	}

	return nil
}

func (supervisor *Supervisor) Stop(id string) error {
	process := supervisor.findByID(id)
	if process == nil {
		return errors.New("failed to find process")
	}

	if !process.Running {
		return errors.New("process already stopped")
	}

	process.Running = false

	if process.Stop != nil {
		process.Stop()
	}

	return nil
}

func (supervisor *Supervisor) Run() {
	if supervisor.Running {
		return
	}

	supervisor.Running = true

	for _, process := range supervisor.Processes {
		go supervisor.processLoop(process)
	}

	select {}
}
