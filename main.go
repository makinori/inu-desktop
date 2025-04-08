package main

import (
	"embed"
	"fmt"
	"net/http"
	"strconv"

	"github.com/charmbracelet/log"
	inu "github.com/makinori/inu-desktop/internal"
)

var (
	//go:embed assets/page.html
	staticContent embed.FS

	mgr *inu.Supervisor = inu.NewSupervisor()
)

func setupFFmpeg() {
	// p1 fastest (lowest)
	// p2 faster (lower)
	// p3 fast (low)
	// p4 medium (default)
	// p5 slow (good)
	// p6 slower (better)
	// p7 slowest (best)

	ffmpegArgs := []string{"-hide_banner", "-nostats", "-re"}

	if inu.IN_CONTAINER {
		ffmpegArgs = append(ffmpegArgs,
			"-video_size",
			fmt.Sprintf("%dx%d", inu.SCREEN_WIDTH, inu.SCREEN_HEIGHT),
			"-framerate", strconv.Itoa(inu.FRAMERATE),
			"-f", "x11grab", "-i", ":0",
		)

	} else {
		log.Warn("using test pattern for ffmpeg")

		const testPattern = true
		if testPattern {
			ffmpegArgs = append(ffmpegArgs,
				"-f", "lavfi", "-i", "testsrc",
				"-sws_flags", "neighbor",
			)
		} else {
			ffmpegArgs = append(ffmpegArgs,
				"-video_size",
				fmt.Sprintf("%dx%d", inu.SCREEN_WIDTH, inu.SCREEN_HEIGHT),
				"-framerate", strconv.Itoa(inu.FRAMERATE),
				"-f", "v4l2", "-i", "/dev/video0",
			)
		}
	}

	// ffmpeg -hide_banner -h encoder=h264_nvenc

	ffmpegArgs = append(ffmpegArgs,
		"-filter:v", fmt.Sprintf("scale=%d:%d", inu.SCREEN_WIDTH, inu.SCREEN_HEIGHT),
		"-pix_fmt", "yuv420p", "-profile:v", "baseline", // doesnt work
		"-c:v", "h264_nvenc", "-b:v", "8000K",
		"-rc", "cbr", "-preset", "p5", "-tune", "ull",
		"-multipass", "qres", "-zerolatency", "1",
		"-g", strconv.Itoa(inu.FRAMERATE/2), "-an", "-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d", inu.LocalRtpPort),
		// ?pkt_size=1316
	)

	mgr.AddSimple(
		"ffmpeg",
		"ffmpeg", ffmpegArgs...,
	)
}

func setupDesktop() {
	mgr.AddSimple(
		"xvfb",
		"Xvfb", ":0", "-screen", "0",
		fmt.Sprintf("%dx%dx24", inu.SCREEN_WIDTH, inu.SCREEN_HEIGHT),
	)

	mgr.AddSimple(
		"dbus",
		"dbus-daemon", "--system", "--nofork", "--print-address",
	)

	mgr.AddSimple(
		"pulseaudio",
		"su", "inu", "-c",
		"dbus-launch pulseaudio --disallow-module-loading --disallow-exit --exit-idle-time=-1",
	)

	mgr.AddSimple(
		"xfce",
		"su", "inu", "-c",
		"dbus-launch xfce4-session --display :0",
	)
}

func main() {
	if !inu.IN_CONTAINER {
		log.Warn("not in container! skipping certain tasks")
	}

	httpMux := http.NewServeMux()

	inu.SetupWebRTC(httpMux)

	httpMux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticContent, "assets/page.html")
	})

	mgr.Add("http", func() {
		log.Infof("public http listening at http://0.0.0.0:%d", inu.WEB_PORT)

		err := http.ListenAndServe(":"+strconv.Itoa(inu.WEB_PORT), httpMux)
		if err != nil {
			log.Error(err)
		}
	})

	if inu.IN_CONTAINER {
		setupDesktop()
	}

	setupFFmpeg()

	mgr.Run()
}
