package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/charmbracelet/log"
)

var (
	//go:embed page.html
	staticContent embed.FS

	mgr *Supervisor = NewSupervisor()
)

func initFFmpeg() {

	// p1 fastest (lowest)
	// p2 faster (lower)
	// p3 fast (low)
	// p4 medium (default)
	// p5 slow (good)
	// p6 slower (better)
	// p7 slowest (best)

	fps := 60
	width := 1920
	height := 1080

	ffmpegArgs := []string{"-hide_banner", "-nostats", "-re"}

	if IN_CONTAINER {
		// x11 grab

	} else {
		const testPattern = true
		if testPattern {
			ffmpegArgs = append(ffmpegArgs,
				"-f", "lavfi", "-i", "testsrc",
				"-sws_flags", "neighbor",
			)
		} else {
			ffmpegArgs = append(ffmpegArgs,
				"-video_size", fmt.Sprintf("%dx%d", width, height),
				"-framerate", strconv.Itoa(fps),
				"-f", "v4l2", "-i", "/dev/video0",
			)
		}
	}

	// ffmpeg -hide_banner -h encoder=h264_nvenc

	ffmpegArgs = append(ffmpegArgs,
		"-filter:v", fmt.Sprintf("scale=%d:%d", width, height),
		"-pix_fmt", "yuv420p", "-profile:v", "baseline", // doesnt work
		"-c:v", "h264_nvenc", "-b:v", "8000K",
		"-rc", "cbr", "-preset", "p5", "-tune", "ull",
		"-multipass", "qres", "-zerolatency", "1",
		"-g", strconv.Itoa(fps/2), "-an", "-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d", localRtpPort),
		// ?pkt_size=1316
	)

	mgr.Add("ffmpeg", func() *exec.Cmd {
		cmd := exec.Command("ffmpeg", ffmpegArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout
		return cmd
	})
}

func initDesktop() {
	fmt.Println("in container lets bootstrap desktop")
}

func main() {
	httpMux = http.NewServeMux()

	initWebRTC(httpMux)

	httpMux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticContent, "page.html")
	})

	log.Infof("public http listening at http://0.0.0.0:%d", WEB_PORT)
	go func() {
		err := http.ListenAndServe(":"+strconv.Itoa(WEB_PORT), httpMux)
		if err != nil {
			panic(err)
		}
	}()

	if IN_CONTAINER {
		initDesktop()
	}

	initFFmpeg()

	mgr.Run()
}
