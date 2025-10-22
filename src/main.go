package src

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/makinori/inu-desktop/src/config"
	"github.com/makinori/inu-desktop/src/inuws"
	"github.com/makinori/inu-desktop/src/supervisor"
	"github.com/makinori/inu-desktop/src/webrtc"
)

var (
	//go:embed assets
	staticContent embed.FS

	processes = supervisor.New()
)

func initGStreamer() {
	videoSrc := "ximagesrc"
	audioSrc := "pulsesrc device=auto_null.monitor"

	if !config.IN_CONTAINER {
		videoSrc = "videotestsrc"
		audioSrc = "audiotestsrc freq=220"
	}

	// https://gstreamer.freedesktop.org/documentation/x264/index.html
	videoEnc := "x264enc " +
		"bitrate=6000 " +
		"pass=cbr " +
		"tune=zerolatency " +
		"speed-preset=veryfast " +
		fmt.Sprintf("key-int-max=%d", config.FRAMERATE)

	if config.USE_NVIDIA {
		// https://gstreamer.freedesktop.org/documentation/nvcodec/nvh264enc.html
		videoEnc = "nvh264enc " +
			"bitrate=6000 " +
			"rc-mode=2 " + // CBR
			"tune=3 " + // Ultra low latency
			"multi-pass=2 " + // Two pass with quarter resolution
			"preset=5 " + // Low Latency, High Performance
			"zerolatency=true " +
			// Number of frames between intra frames
			fmt.Sprintf("gop-size=%d", config.FRAMERATE)

		slog.Info("using nvidia for video encoding")
	} else {
		slog.Info("using cpu for video encoding")
	}

	videoPipeline := []string{
		videoSrc,
		fmt.Sprintf(
			"video/x-raw,width=%d,height=%d,framerate=%d/1", // ,format=NV12
			config.SCREEN_WIDTH, config.SCREEN_HEIGHT, config.FRAMERATE,
		),
		"videoconvert",
		videoEnc,
		"h264parse config-interval=-1",
		"video/x-h264,stream-format=byte-stream,profile=constrained-baseline",
		"rtph264pay",
		fmt.Sprintf("udpsink host=127.0.0.1 port=%d", webrtc.LocalRtpVideoPort),
	}

	// https://wiki.xiph.org/Opus_Recommended_Settings
	audioPipeline := []string{
		audioSrc,
		"audioconvert",
		"opusenc bitrate=320000",
		"rtpopuspay",
		fmt.Sprintf("udpsink host=127.0.0.1 port=%d", webrtc.LocalRtpAudioPort),
	}

	videoCommand := "gst-launch-1.0 --no-position " +
		strings.Join(videoPipeline, " ! ")

	audioCommand := "gst-launch-1.0 --no-position " +
		strings.Join(audioPipeline, " ! ")

	// TODO: set PULSE_LATENCY_MSEC really low?

	processes.AddCommand(supervisor.Command{
		ID:          "gst-video",
		Command:     "sh",
		Args:        []string{"-c", videoCommand},
		NoAutoStart: true,
	})

	if config.IN_CONTAINER {
		processes.AddCommand(supervisor.Command{
			ID:          "gst-audio",
			Command:     "su",
			Args:        []string{"inu", "-c", audioCommand},
			NoAutoStart: true,
		})
	} else {
		processes.AddCommand(supervisor.Command{
			ID:          "gst-audio",
			Command:     "sh",
			Args:        []string{"-c", audioCommand},
			NoAutoStart: true,
		})
	}
}

func initGlobalEnv() {
	if config.USE_NVIDIA {
		// os.Setenv("GBM_BACKEND", "nvidia-drm")
		// os.Setenv("__GLX_VENDOR_LIBRARY_NAME", "nvidia")
		os.Setenv("LIBVA_DRIVER_NAME", "nvidia") // TODO: does this work?
		os.Setenv("VGL_DISPLAY", "egl")
	}
}

func initDesktop() {
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

	processes.AddCommand(supervisor.Command{
		ID:      "pulseaudio",
		Command: "su",
		Args: []string{
			"inu", "-c",
			"dbus-launch pulseaudio --disallow-module-loading --disallow-exit " +
				"--exit-idle-time=-1",
		},
	})

	xfceCommand := "dbus-launch xfce4-session --display :0"

	if config.USE_NVIDIA {
		xfceCommand = "vglrun " + xfceCommand
	}

	processes.AddCommand(supervisor.Command{
		ID:      "xfce",
		Command: "su",
		Args:    []string{"inu", "-c", xfceCommand},
	})

	// need systemd-login
	// processes.AddCommand(supervisor.Command{
	// 	ID:      "gnome",
	// 	Command: "su",
	// 	Args: []string{
	// 		"inu", "-c",
	// 		"dbus-launch gnome-shell --x11 -d :0",
	// 	},
	// })
}

func Main() {
	if !config.IN_CONTAINER {
		slog.Warn("not in container! skipping certain tasks")
	}

	httpMux := http.NewServeMux()

	webrtc.Init(httpMux)

	inuws.Init(httpMux, &webrtc.ViewerCount, webrtc.ViewerCountSignal)

	if config.IN_CONTAINER {
		assets, err := fs.Sub(staticContent, "assets")
		if err != nil {
			panic(err)
		}
		httpMux.Handle("/", http.FileServerFS(assets))
	} else {
		httpMux.Handle("/", http.FileServer(http.Dir("src/assets/")))
	}

	processes.AddSimple("http", func() error {
		slog.Info(
			"public http listening at http://0.0.0.0:" +
				strconv.Itoa(config.WEB_PORT),
		)

		err := http.ListenAndServe(":"+strconv.Itoa(config.WEB_PORT), httpMux)
		if err != nil {
			slog.Error(err.Error())
		}

		return nil
	})

	initGlobalEnv()

	if config.IN_CONTAINER {
		initDesktop()
	}

	initGStreamer()

	webrtc.ViewerCountSignal.AddListener(
		func(ctx context.Context, value uint32) {
			if value == 0 {
				processes.Stop("gst-video")
				processes.Stop("gst-audio")
			} else {
				processes.Start("gst-video")
				processes.Start("gst-audio")
			}
		},
	)

	processes.Run()
}
