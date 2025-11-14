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
	tailOpts  []Option
}

var shutdownCh = make(chan struct{})

// Shutdown signals all SSE handlers to shut down
// This will cause all active watchers to terminate gracefully.
func Shutdown() {
	close(shutdownCh)
}

func Handler(cutPrefix string, filepath string) http.Handler {
	return handler{
		Filename:  filepath,
		CutPrefix: cutPrefix,
		fsServer:  http.FileServerFS(staticFS),
		tailOpts: []Option{
			WithPollInterval(500 * time.Millisecond),
			WithBufferSize(1000),
			WithPlugins(NewColoring("default")),
		},
	}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "watch.stream") {
		h.serveWatcher(w, r)
	} else {
		h.serveStatic(w, r)
	}
}

func (h handler) serveWatcher(w http.ResponseWriter, r *http.Request) {
	if h.Filename == "" {
		http.Error(w, "Filename not configured", http.StatusNotImplemented)
		return
	}

	opts := append([]Option{}, h.tailOpts...)

	filterParam := r.URL.Query().Get("filter")
	filters := strings.Split(filterParam, "||")
	for _, filter := range filters {
		splits := strings.Split(filter, "&&")
		toks := make([]string, 0, len(splits))
		for _, tok := range splits {
			tok = strings.TrimSpace(tok)
			if tok != "" {
				toks = append(toks, tok)
			}
		}
		if len(toks) > 0 {
			opts = append(opts, WithPattern(toks...))
		}
	}

	tail := New(h.Filename, opts...)
	if err := tail.Start(); err != nil {
		http.Error(w, "Failed to start watcher", http.StatusInternalServerError)
		return
	}
	defer tail.Stop()

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	rc.Flush()

	flushTicker := time.NewTicker(1 * time.Second)
	defer flushTicker.Stop()
	for {
		select {
		case <-flushTicker.C:
			rc.Flush()
		case line := <-tail.Lines():
			fmt.Fprintf(w, "data: %s\n\n", line)
		case <-r.Context().Done():
			return
		case <-shutdownCh:
			return
		}
	}
}

//go:embed static/*
var staticFS embed.FS

func (h handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = "static/" + strings.TrimPrefix(r.URL.Path, h.CutPrefix)
	h.fsServer.ServeHTTP(w, r)
}
