package src

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/makinori/inu-desktop/src/config"
	"github.com/makinori/inu-desktop/src/inuwebrtc"
	"github.com/makinori/inu-desktop/src/inuws"
	"github.com/makinori/inu-desktop/src/supervisor"
)

var (
	//go:embed assets
	staticContent embed.FS

	processes = supervisor.New()
)

func Main() {
	if !config.IN_CONTAINER {
		slog.Warn("not in container! skipping certain tasks")
	}

	httpMux := http.NewServeMux()

	inuwebrtc.Init(httpMux)

	inuws.Init(httpMux, &inuwebrtc.ViewerCount, inuwebrtc.ViewerCountSignal)

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
			"public http listening at " + strconv.Itoa(config.WEB_PORT),
		)

		err := http.ListenAndServe(":"+strconv.Itoa(config.WEB_PORT), httpMux)
		if err != nil {
			slog.Error(err.Error())
		}

		return nil
	})

	if config.IN_CONTAINER {
		initDesktop()
	}

	initGStreamer()

	inuwebrtc.ViewerCountSignal.AddListener(
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
