package tailer

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type handler struct {
	Filename  string
	CutPrefix string
	fsServer  http.Handler
}

func Handler(cutPrefix string, filepath string) http.Handler {
	return handler{
		Filename:  filepath,
		CutPrefix: cutPrefix,
		fsServer:  http.FileServerFS(staticFS),
	}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "watch.stream") {
		watch := NewWatcher(string(h.Filename))
		watch.ServeHTTP(w, r)
		return
	}

	r.URL.Path = "static/" + strings.TrimPrefix(r.URL.Path, h.CutPrefix)
	h.fsServer.ServeHTTP(w, r)
}

//go:embed static/*
var staticFS embed.FS

func NewWatcher(filename string) http.Handler {
	return &Watcher{
		filename: filename,
	}
}

type Watcher struct {
	filename string
	tail     *Tail
}

var _ http.Handler = (*Watcher)(nil)

func (lw *Watcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if lw.filename == "" {
		http.Error(w, "Log filename not configured", http.StatusNotImplemented)
		return
	}

	lw.tail = New(lw.filename, WithPollInterval(500*time.Millisecond))
	if err := lw.tail.Start(); err != nil {
		http.Error(w, "Failed to start log watcher", http.StatusInternalServerError)
		return
	}
	defer lw.tail.Stop()

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	rc.Flush()

	for {
		select {
		case line := <-lw.tail.Lines():
			fmt.Fprintf(w, "data: %s\n\n", line)
			rc.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
