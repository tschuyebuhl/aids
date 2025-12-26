package httpx

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type hookedResponseWriter struct {
	http.ResponseWriter
	got404 bool
}

func (hrw *hookedResponseWriter) WriteHeader(status int) {
	if status == http.StatusNotFound {
		// Don't actually write the 404 header, just set a flag.
		hrw.got404 = true
	} else {
		hrw.ResponseWriter.WriteHeader(status)
	}
}

func (hrw *hookedResponseWriter) Write(p []byte) (int, error) {
	if hrw.got404 {
		// No-op, but pretend that we wrote len(p) bytes to the writer.
		return len(p), nil
	}

	return hrw.ResponseWriter.Write(p)
}
func Intercept404(handler, on404 http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hookedWriter := &hookedResponseWriter{ResponseWriter: w}
		handler.ServeHTTP(hookedWriter, r)

		if hookedWriter.got404 {
			on404.ServeHTTP(w, r)
		}
	})
}
func ServeFileContents(file string, files http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Restrict only to instances where the browser is looking for an HTML file
		if !strings.Contains(r.Header.Get("Accept"), "text/html") {
			w.WriteHeader(http.StatusNotFound)
			if _, err := fmt.Fprint(w, "404 not found"); err != nil {
				slog.Error("error writing 404 response", "err", err)
			}

			return
		}

		// Open the file and return its contents using http.ServeContent
		index, err := files.Open(file)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			if _, err := fmt.Fprintf(w, "%s not found", file); err != nil {
				slog.Error("error writing missing file response", "file", file, "err", err)
			}

			return
		}

		fi, err := index.Stat()
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			if _, err := fmt.Fprintf(w, "%s not found", file); err != nil {
				slog.Error("error writing missing file response", "file", file, "err", err)
			}

			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), index)
	}
}

func DevProxy(frontendAddress string) *httputil.ReverseProxy {
	proxyURL, _ := url.Parse(frontendAddress)
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)

	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.Header.Get("Upgrade") == "websocket" {
			resp.Header.Set("Connection", "upgrade")
		}
		return nil
	}
	return proxy
}

func RunEmbeddedApp(appRoot string, embeddedFS embed.FS, mux *http.ServeMux) {
	frontendFS, err := fs.Sub(embeddedFS, "frontend/dist")
	if err != nil {
		panic(err)
	}
	httpFS := http.FS(frontendFS)
	fileServer := http.FileServer(httpFS)
	serveIndex := ServeFileContents("index.html", httpFS)

	mux.Handle(appRoot, Intercept404(fileServer, serveIndex))
}
