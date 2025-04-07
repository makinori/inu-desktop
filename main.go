package main

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/log"
)

func initFFmpeg(keepAlive chan bool) {
	// ffmpeg -hide_banner -h encoder=h264_nvenc

	// p1 fastest (lowest)
	// p2 faster (lower)
	// p3 fast (low)
	// p4 medium (default)
	// p5 slow (good)
	// p6 slower (better)
	// p7 slowest (best)

	ffmpegArgs := []string{
		"-nostats", "-re", "-f", "v4l2",
		"-video_size", "1920x1080", "-framerate", "60", "-i", "/dev/video0",
		"-c:v", "h264_nvenc", "-b:v", "8000K",
		"-rc", "cbr", "-preset", "p5", "-tune", "ull", "-profile:v", "baseline",
		"-multipass", "qres", "-zerolatency", "1",
		"-g", "30", "-an", "-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d", localRtpPort),
		// ?pkt_size=1316
	}

	ffmpegCmd := exec.Command("ffmpeg", ffmpegArgs...)
	// ffmpegCmd.Stdout = os.Stdout
	// ffmpegCmd.Stderr = os.Stdout

	log.Info("starting ffmpeg...")

	err := ffmpegCmd.Run()
	if err != nil {
		log.Error(err)
	}

	<-keepAlive
}

func main() {
	keepAlive := make(chan bool)

	initWebRTC()

	initFFmpeg(keepAlive)

	<-keepAlive
}
