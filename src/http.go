package src

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/makinori/inu-desktop/src/config"
)

func initWeb(httpMux *http.ServeMux) {
	var assetsFS fs.FS
	if config.IN_CONTAINER {
		var err error
		assetsFS, err = fs.Sub(staticContent, "assets")
		if err != nil {
			panic(err)
		}
	} else {
		assetsFS = os.DirFS("src/assets/")
	}

	httpMux.Handle("/", http.FileServerFS(assetsFS))

	// hastily written, but doing this to avoid js on repo on github
	// perhaps use optimized functions from maki.cafe

	httpMux.HandleFunc("/js/{filename}", func(w http.ResponseWriter, r *http.Request) {
		filename := r.PathValue("filename")

		gzipData, err := fs.ReadFile(assetsFS, "js/"+filename+".gz")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Add("Content-Encoding", "gzip")
			http.ServeContent(
				w, r, filename, time.Now(), bytes.NewReader(gzipData),
			)
			return
		}

		// no gzip supported on client

		reader, err := gzip.NewReader(bytes.NewReader(gzipData))
		if err != nil {
			slog.Error("gzip", "err", err.Error())
			http.Error(w, "gzip error", http.StatusInternalServerError)
			return
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			slog.Error("gzip read", "err", err.Error())
			http.Error(w, "gzip read error", http.StatusInternalServerError)
			return
		}

		http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(data))
	})
}
