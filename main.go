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

func setupFFmpeg() {
	ffmpegArgs := []string{"-hide_banner", "-nostats", "-re"}

	ffmpegVideoArgs := ffmpegArgs
	ffmpegAudioArgs := ffmpegArgs

	if IN_CONTAINER {
		ffmpegVideoArgs = append(ffmpegVideoArgs,
			"-video_size",
			fmt.Sprintf("%dx%d", SCREEN_WIDTH, SCREEN_HEIGHT),
			"-framerate", strconv.Itoa(FRAMERATE),
			"-f", "x11grab", "-i", ":0",
		)
	} else {
		log.Warn("using test pattern for ffmpeg")

		const testPattern = true
		if testPattern {
			ffmpegVideoArgs = append(ffmpegVideoArgs,
				"-f", "lavfi", "-i", "testsrc",
				"-sws_flags", "neighbor",
			)
		} else {
			ffmpegVideoArgs = append(ffmpegVideoArgs,
				"-video_size",
				fmt.Sprintf("%dx%d", SCREEN_WIDTH, SCREEN_HEIGHT),
				"-framerate", strconv.Itoa(FRAMERATE),
				"-f", "v4l2", "-i", "/dev/video0",
			)
		}
	}

	// p1 fastest (lowest)
	// p2 faster (lower)
	// p3 fast (low)
	// p4 medium (default)
	// p5 slow (good)
	// p6 slower (better)
	// p7 slowest (best)

	// ffmpeg -hide_banner -h encoder=h264_nvenc

	ffmpegVideoArgs = append(ffmpegVideoArgs,
		"-filter:v", fmt.Sprintf("scale=%d:%d", SCREEN_WIDTH, SCREEN_HEIGHT),
		"-pix_fmt", "yuv420p", "-profile:v", "baseline",
		"-c:v", "h264_nvenc", "-b:v", "8000K",
		"-rc", "cbr", "-preset", "p5", "-tune", "ull",
		"-multipass", "qres", "-zerolatency", "1",
		"-g", strconv.Itoa(FRAMERATE/2), "-an", "-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d", LocalRtpVideoPort),
		// ?pkt_size=1316
	)

	mgr.AddSimple(
		"ffmpeg-video",
		"ffmpeg", ffmpegVideoArgs...,
	)

	// audio

	if IN_CONTAINER {
		ffmpegAudioArgs = append(ffmpegAudioArgs,
			"-f", "pulse", "-i", "auto_null.monitor",
		)
	} else {
		ffmpegAudioArgs = append(ffmpegAudioArgs,
			"-f", "lavfi", "-i", "sine=f=440:r=48000",
		)
	}

	// https://ffmpeg.org/ffmpeg-codecs.html#libopus-1
	// https://github.com/pion/webrtc/issues/1514

	// TODO: stream gets laggy after a while still

	ffmpegAudioArgs = append(ffmpegAudioArgs,
		"-c:a", "libopus", "-b:a", "128K", "-vbr", "on",
		"-compression_level", "10", "-frame_duration", "20",
		"-application", "lowdelay", "-sample_fmt", "s16", "-ssrc", "1",
		"-vn",
		// this might be the cause?
		// "-af", "adelay=0:all=true", "-async", "1",
		"-payload_type", "111", "-f", "rtp", "-max_delay", "0",
		fmt.Sprintf("rtp://127.0.0.1:%d", LocalRtpAudioPort),
	)

	mgr.AddSimple(
		"ffmpeg-audio",
		"su", "inu", "-c",
		"ffmpeg "+strings.Join(ffmpegAudioArgs, " "),
	)
}

func setupDesktop() {
	mgr.AddSimple(
		"xvfb",
		"Xvfb", ":0", "-screen", "0",
		fmt.Sprintf("%dx%dx24", SCREEN_WIDTH, SCREEN_HEIGHT),
	)

	mgr.AddSimple(
		"dbus",
		"su", "inu", "-c",
		"dbus-daemon --session --nofork --nopidfile",
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

	setupFFmpeg()

	mgr.Run()
}
