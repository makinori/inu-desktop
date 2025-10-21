package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	. "github.com/makinori/inu-desktop/internal"
)

var (
	//go:embed assets
	staticContent embed.FS

	mgr *Supervisor = NewSupervisor()
)

func setupGStreamer() {
	videoSrc := "ximagesrc"
	audioSrc := "pulsesrc device=auto_null.monitor"

	if !IN_CONTAINER {
		videoSrc = "videotestsrc"
		audioSrc = "audiotestsrc freq=220"
	}

	// https://gstreamer.freedesktop.org/documentation/x264/index.html
	videoEnc := "x264enc " +
		"bitrate=6000 " +
		"pass=cbr " +
		"tune=zerolatency " +
		"speed-preset=veryfast " +
		fmt.Sprintf("key-int-max=%d", FRAMERATE)

	if USE_NVIDIA {
		// https://gstreamer.freedesktop.org/documentation/nvcodec/nvh264enc.html
		videoEnc = "nvh264enc " +
			"bitrate=6000 " +
			"rc-mode=2 " + // CBR
			"tune=3 " + // Ultra low latency
			"multi-pass=2 " + // Two pass with quarter resolution
			"preset=5 " + // Low Latency, High Performance
			"zerolatency=true " +
			// Number of frames between intra frames
			fmt.Sprintf("gop-size=%d", FRAMERATE)
	}

	videoPipeline := []string{
		videoSrc,
		fmt.Sprintf(
			"video/x-raw,width=%d,height=%d,framerate=%d/1", // ,format=NV12
			SCREEN_WIDTH, SCREEN_HEIGHT, FRAMERATE,
		),
		"videoconvert",
		videoEnc,
		"h264parse config-interval=-1",
		"video/x-h264,stream-format=byte-stream,profile=constrained-baseline",
		"rtph264pay",
		fmt.Sprintf("udpsink host=127.0.0.1 port=%d", LocalRtpVideoPort),
	}

	// https://wiki.xiph.org/Opus_Recommended_Settings
	audioPipeline := []string{
		audioSrc,
		"audioconvert",
		"opusenc bitrate=320000",
		"rtpopuspay",
		fmt.Sprintf("udpsink host=127.0.0.1 port=%d", LocalRtpAudioPort),
	}

	videoCommand := "gst-launch-1.0 --no-position " +
		strings.Join(videoPipeline, " ! ")

	audioCommand := "gst-launch-1.0 --no-position " +
		strings.Join(audioPipeline, " ! ")

	// set PULSE_LATENCY_MSEC really low?

	if IN_CONTAINER {
		mgr.AddSimple("gst-video", "sh", "-c", videoCommand)
		mgr.AddSimple("gst-audio", "su", "inu", "-c", audioCommand)
	} else {
		mgr.AddSimple("gst-video", "sh", "-c", videoCommand)
		mgr.AddSimple("gst-audio", "sh", "-c", audioCommand)
	}
}

func setupDesktop() {
	mgr.AddSimple(
		"xvfb",
		"Xvfb", ":0", "-screen", "0",
		fmt.Sprintf("%dx%dx24", SCREEN_WIDTH, SCREEN_HEIGHT),
	)

	// userEnv := []string{
	// 	"XDG_RUNTIME_DIR=/run/user/1000",
	// 	"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus",
	// }

	mgr.AddSimple(
		"dbus",
		"dbus-daemon", "--system", "--nofork", "--nopidfile",
		// "su", "inu", "-c",
		// "dbus-daemon --session --nofork --nopidfile",
		// doesnt work DBUS_SESSION_BUS_ADDRESS is still tmp
		// "--address=unix:path=/run/user/1000/bus",
		// XDG_RUNTIME_DIR also doesnt get set
	)

	mgr.AddSimple(
		"pulseaudio",
		"su", "inu", "-c",
		"dbus-launch pulseaudio --disallow-module-loading --disallow-exit "+
			"--exit-idle-time=-1",
	)

	mgr.AddSimple(
		"xfce",
		"su", "inu", "-c",
		"dbus-launch xfce4-session --display :0",
	)

	// need systemd-login
	// mgr.AddSimple(
	// 	"gnome",
	// 	"su", "inu", "-c",
	// 	"dbus-launch gnome-shell --x11 -d :0",
	// )
}

func main() {
	if !IN_CONTAINER {
		log.Warn("not in container! skipping certain tasks")
	}

	httpMux := http.NewServeMux()

	SetupWebRTC(httpMux)

	if IN_CONTAINER {
		SetupWebSocket(httpMux)

		assets, err := fs.Sub(staticContent, "assets")
		if err != nil {
			panic(err)
		}
		httpMux.Handle("/", http.FileServerFS(assets))
	} else {
		httpMux.Handle("/", http.FileServer(http.Dir("assets/")))
	}

	mgr.Add("http", func() {
		log.Infof("public http listening at http://0.0.0.0:%d", WEB_PORT)

		err := http.ListenAndServe(":"+strconv.Itoa(WEB_PORT), httpMux)
		if err != nil {
			log.Error(err)
		}
	})

	if IN_CONTAINER {
		setupDesktop()
	}

	setupGStreamer()

	mgr.Run()
}
