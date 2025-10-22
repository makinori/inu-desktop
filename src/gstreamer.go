package src

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/makinori/inu-desktop/src/config"
	"github.com/makinori/inu-desktop/src/inuwebrtc"
	"github.com/makinori/inu-desktop/src/supervisor"
)

func initGStreamer() {
	videoSrc := "ximagesrc use-damage=false"
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
		fmt.Sprintf("udpsink host=127.0.0.1 port=%d", inuwebrtc.LocalRtpVideoPort),
	}

	// https://wiki.xiph.org/Opus_Recommended_Settings
	audioPipeline := []string{
		audioSrc,
		"audioconvert",
		"opusenc bitrate=320000",
		"rtpopuspay",
		fmt.Sprintf("udpsink host=127.0.0.1 port=%d", inuwebrtc.LocalRtpAudioPort),
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
