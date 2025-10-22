package src

import (
	"fmt"
	"os"

	"github.com/makinori/inu-desktop/src/config"
	"github.com/makinori/inu-desktop/src/supervisor"
)

func initDesktop() {
	if config.USE_NVIDIA {
		// os.Setenv("GBM_BACKEND", "nvidia-drm")
		// os.Setenv("__GLX_VENDOR_LIBRARY_NAME", "nvidia")
		os.Setenv("LIBVA_DRIVER_NAME", "nvidia") // TODO: does this work?
		os.Setenv("VGL_DISPLAY", "egl")
	}

	xvfbCommand := "Xvfb :0 -screen 0 " +
		fmt.Sprintf("%dx%dx24", config.SCREEN_WIDTH, config.SCREEN_HEIGHT)

	if config.USE_NVIDIA {
		xvfbCommand = "vglrun " + xvfbCommand
	}

	processes.AddCommand(supervisor.Command{
		ID:      "xvfb",
		Command: "sh",
		Args:    []string{"-c", xvfbCommand},
	})

	// Xorg :0.0 -config .conf -noreset -nolisten tcp

	// userEnv := []string{
	// 	"XDG_RUNTIME_DIR=/run/user/1000",
	// 	"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus",
	// }

	processes.AddCommand(supervisor.Command{
		ID:      "dbus",
		Command: "dbus-daemon",
		Args:    []string{"--system", "--nofork", "--nopidfile"},
		// "su", "inu", "-c",
		// "dbus-daemon --session --nofork --nopidfile",
		// doesnt work DBUS_SESSION_BUS_ADDRESS is still tmp
		// "--address=unix:path=/run/user/1000/bus",
		// XDG_RUNTIME_DIR also doesnt get set
	})

	runAsInu := func(id string, command string, withVGL bool) {
		if withVGL && config.USE_NVIDIA {
			command = "vglrun " + command
		}
		command = "dbus-launch " + command
		processes.AddCommand(supervisor.Command{
			ID:      id,
			Dir:     "/home/inu",
			Command: "su",
			Args:    []string{"inu", "-c", command},
		})
	}

	runAsInu(
		"pulseaudio",
		"pulseaudio --disallow-module-loading --disallow-exit "+
			"--exit-idle-time=-1",
		false,
	)

	// runAsInu("xfce", "xfce4-session --display :0", true)

	runAsInu("openbox", "openbox-session", true)
}
