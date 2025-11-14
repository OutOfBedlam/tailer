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

func Handler(cutPrefix string, filepath string) http.Handler {
	return handler{
		Filename:  filepath,
		CutPrefix: cutPrefix,
		fsServer:  http.FileServerFS(staticFS),
		tailOpts: []Option{
			WithPollInterval(500 * time.Millisecond),
			WithBufferSize(1000),
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

	tail := New(h.Filename, h.tailOpts...)
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
			fmt.Fprintf(w, "data: %s\n\n", colors(line))
		case <-r.Context().Done():
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

// colors formats a line for xterm js coloring
// For now, it just converts TRACE, DEBUG, INFO, WARN, ERROR to colors
func colors(line string) string {
	// Replace log levels with colored versions
	line = strings.ReplaceAll(line, "TRACE", colorTrace+"TRACE"+colorReset)
	// line = strings.ReplaceAll(line, "DEBUG", colorDebug+"DEBUG"+colorReset)
	line = strings.ReplaceAll(line, "INFO", colorInfo+"INFO"+colorReset)
	line = strings.ReplaceAll(line, "WARN", colorWarn+"WARN"+colorReset)
	line = strings.ReplaceAll(line, "ERROR", colorError+"ERROR"+colorReset)

	return line
}

// ANSI color codes for xterm.js
const (
	// colorCyan  = "\033[36m" // Cyan
	// colorGreen  = "\033[32m" // Green
	colorReset = "\033[0m"
	colorTrace = "\033[37m" // Light gray
	colorInfo  = "\033[34m" // Blue
	colorWarn  = "\033[33m" // Yellow
	colorError = "\033[31m" // Red
)
