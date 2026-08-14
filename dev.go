package main

import (
	"bytes"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const liveReloadScript = `<script>
(() => {
    const events = new EventSource("/__dev/reload");
    events.addEventListener("reload", () => window.location.reload());
})();
</script>`

type fileVersion struct {
	modified int64
	size     int64
}

type liveReloader struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func newLiveReloader() *liveReloader {
	return &liveReloader{clients: make(map[chan struct{}]struct{})}
}

func (lr *liveReloader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")

	changes := make(chan struct{}, 1)
	lr.mu.Lock()
	lr.clients[changes] = struct{}{}
	lr.mu.Unlock()
	defer func() {
		lr.mu.Lock()
		delete(lr.clients, changes)
		lr.mu.Unlock()
	}()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-changes:
			fmt.Fprint(w, "event: reload\ndata: changed\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (lr *liveReloader) watch(paths []string) {
	previous := snapshot(paths)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		current := snapshot(paths)
		if maps.Equal(previous, current) {
			continue
		}

		previous = current
		log.Println("Development files changed; reloading browsers")
		lr.notify()
	}
}

func (lr *liveReloader) notify() {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	for client := range lr.clients {
		select {
		case client <- struct{}{}:
		default:
		}
	}
}

func snapshot(paths []string) map[string]fileVersion {
	files := make(map[string]fileVersion)
	for _, root := range paths {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}

			info, err := entry.Info()
			if err == nil {
				files[path] = fileVersion{
					modified: info.ModTime().UnixNano(),
					size:     info.Size(),
				}
			}
			return nil
		})
	}
	return files
}

type bufferedResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header)}
}

func (w *bufferedResponse) Header() http.Header {
	return w.header
}

func (w *bufferedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *bufferedResponse) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}

func injectLiveReload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") || r.URL.Path == "/__dev/reload" {
			next.ServeHTTP(w, r)
			return
		}

		response := newBufferedResponse()
		next.ServeHTTP(response, r)

		body := response.body.Bytes()
		if response.status >= 200 && response.status < 300 {
			body = bytes.Replace(body, []byte("</body>"), []byte(liveReloadScript+"\n</body>"), 1)
		}

		for key, values := range response.header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.Header().Del("Content-Length")
		w.Header().Set("Cache-Control", "no-store")
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(body))
		}

		status := response.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}
