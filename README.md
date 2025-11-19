# Tailer

A cross-platform Go library for tailing files, similar to the Unix `tail -F` command. It monitors files for changes and supports log rotation, truncation, and pattern filtering.

## Features

- 📝 **Real-time file tailing**: Monitor files for new content as they grow
- 🔄 **Log rotation detection**: Automatically detects when files are rotated and follows the new file
- ✂️ **Truncation handling**: Detects when files are truncated and starts from the beginning
- 🔍 **Pattern filtering**: Filter lines using regular expressions (grep-like functionality)
- 🌐 **Web interface**: Built-in HTTP handler with SSE (Server-Sent Events) for browser-based tailing
- 🖥️ **Terminal UI**: Includes xterm.js-based web terminal with syntax highlighting
- 🪟 **Cross-platform**: Works on Windows, Linux, macOS, and BSD systems
- ⚡ **Efficient**: Uses polling with configurable intervals
- 🎯 **Flexible**: Read last N lines before starting to tail

## Installation

```bash
go get github.com/OutOfBedlam/tailer
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/OutOfBedlam/tailer"
)

func main() {
    // Create a new tailer
    tail := tailer.New("/var/log/app.log")
    
    // Start tailing
    if err := tail.Start(); err != nil {
        panic(err)
    }
    defer tail.Stop()
    
    // Read lines from the channel
    for line := range tail.Lines() {
        fmt.Println(line)
    }
}
```

### Custom Configuration

```go
tail := tailer.New("/var/log/app.log",
    tailer.WithPollInterval(500*time.Millisecond),  // Check file every 500ms
    tailer.WithBufferSize(200),                     // Channel buffer size
    tailer.WithLast(20),                           // Show last 20 lines on start
)
```

### Pattern Filtering (Grep)

Filter lines using regular expressions. Multiple patterns can be specified:

```go
// Show only error and warning lines
tail := tailer.New("/var/log/app.log",
    tailer.WithPattern("error", "warning"),  // Lines matching both "error" AND "warning"
    tailer.WithPattern("fatal"),             // OR lines matching "fatal"
)

if err := tail.Start(); err != nil {
    panic(err)
}
defer tail.Stop()

for line := range tail.Lines() {
    fmt.Println(line)  // Only lines matching the patterns
}
```

Pattern groups work as follows:
- Within a `WithPattern()` call, all patterns must match (AND logic)
- Multiple `WithPattern()` calls are OR'ed together
- Example: `WithPattern("error", "thing")` matches lines containing both "error" AND "thing"
- Example: Multiple calls like `WithPattern("error")` and `WithPattern("warning")` match lines with "error" OR "warning"

### Complete Example with Timeout

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/OutOfBedlam/tailer"
)

func main() {
    tail := tailer.New("/var/log/app.log",
        tailer.WithPollInterval(100*time.Millisecond),
        tailer.WithLast(10),
        tailer.WithPattern("ERROR"),
    )
    
    if err := tail.Start(); err != nil {
        panic(err)
    }
    defer tail.Stop()
    
    timeout := time.After(30 * time.Second)
    
    for {
        select {
        case line := <-tail.Lines():
            fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), line)
        case <-timeout:
            fmt.Println("Tailing complete")
            return
        }
    }
}
```

### Web-Based Tailing with SSE

The package includes a built-in HTTP handler that provides real-time log tailing through Server-Sent Events (SSE) with a beautiful web terminal interface.

#### Simple HTTP Server Example

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/OutOfBedlam/tailer"
)

func main() {
    // Create handler for tailing /var/log/app.log
    // The first parameter is the URL prefix to strip
    handler := tailer.NewHandler("/tail/", "/var/log/app.log")
    
    // Mount the handler
    http.Handle("/tail/", handler)
    
    log.Println("Server starting on http://localhost:8080")
    log.Println("Open http://localhost:8080/tail/ in your browser")
    
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal(err)
    }
}
```

#### Advanced HTTP Server with Multiple Files

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/OutOfBedlam/tailer"
)

func main() {
    mux := http.NewServeMux()
    
    // Tail application logs
    mux.Handle("/logs/app/", tailer.NewHandler("/logs/app/", "/var/log/myapp.log"))
    
    // Tail access logs
    mux.Handle("/logs/access/", tailer.NewHandler("/logs/access/", "/var/log/access.log"))
    
    // Tail error logs
    mux.Handle("/logs/error/", tailer.NewHandler("/logs/error/", "/var/log/error.log"))
    
    log.Println("Multi-log viewer starting on :8080")
    log.Println("Available endpoints:")
    log.Println("  - http://localhost:8080/logs/app/")
    log.Println("  - http://localhost:8080/logs/access/")
    log.Println("  - http://localhost:8080/logs/error/")
    
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatal(err)
    }
}
```

#### Graceful Shutdown

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/OutOfBedlam/tailer"
)

func main() {
    handler := tailer.NewHandler("/", "/var/log/app.log")
    
    server := &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }
    
    // Start server in goroutine
    go func() {
        log.Println("Server starting on :8080")
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down server...")
    
    // Signal all SSE connections to close
    tailer.Shutdown()
    
    // Gracefully shutdown HTTP server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }
    
    log.Println("Server stopped")
}
```

#### Web Interface Features

The built-in web interface includes:
- **xterm.js terminal**: Full-featured terminal emulator in the browser
- **Syntax highlighting**: Automatic colorization of log levels (DEBUG, INFO, WARN, ERROR)
- **Real-time updates**: Server-Sent Events (SSE) push new lines instantly
- **Filter support**: Query parameter filtering with AND/OR logic
- **Responsive design**: Works on desktop and mobile browsers
- **Auto-scrolling**: Terminal automatically scrolls to show new content

#### URL Filter Parameters

You can filter log lines using URL query parameters:

```bash
# Filter for lines containing "error"
http://localhost:8080/tail/?filter=error

# Filter for lines containing both "error" AND "database" (AND logic with &&)
http://localhost:8080/tail/?filter=error&&database

# Filter for lines with "error" OR "warning" (OR logic with ||)
http://localhost:8080/tail/?filter=error||warning

# Complex filter: (error AND database) OR (warning AND timeout)
http://localhost:8080/tail/?filter=error&&database||warning&&timeout
```

Filter syntax:
- `&&` = AND operator (all patterns must match)
- `||` = OR operator (any pattern group can match)
- Patterns are regular expressions

## API Reference

### Types

#### `type Tail`

The main tailer instance that monitors a file.

### Functions

#### `New(filepath string, opts ...Option) *Tail`

Creates a new Tail instance for the specified file path.

**Default values:**
- Poll interval: 1 second
- Buffer size: 100 lines
- Last N lines: 10

#### `(*Tail) Start() error`

Starts tailing the file. Reads the last N lines (configurable) and then monitors for new content.

#### `(*Tail) Stop() error`

Stops tailing and closes the file. This method waits for the internal goroutine to finish before returning.

#### `(*Tail) Lines() <-chan string`

Returns a read-only channel that outputs new lines from the file.

#### `NewHandler(cutPrefix string, filepath string, opts...Option) tailer.Handler`

Creates an HTTP handler that provides web-based log tailing using Server-Sent Events (SSE).

**Parameters:**
- `cutPrefix`: The URL prefix to strip from incoming requests (e.g., "/logs/app/")
- `filepath`: The absolute path to the file to tail

**Returns:** An `http.Handler` that serves:
- A web interface at the base URL (using embedded xterm.js terminal)
- An SSE stream at `{baseURL}/watch.stream` for real-time log updates

The handler automatically:
- Polls the file every 500ms for changes
- Uses a buffer size of 1000 lines
- Colorizes log levels (DEBUG, INFO, WARN, ERROR)
- Supports URL query parameter filtering via `?filter=`

**Example:**
```go
handler := tailer.NewHandler("/logs/", "/var/log/app.log")
http.Handle("/logs/", handler)
```

#### `Shutdown()`

Signals all active SSE handlers to shut down gracefully. Call this before stopping your HTTP server to cleanly close all open connections.

**Example:**
```go
tailer.Shutdown()
server.Shutdown(context.Background())
```

### Options

#### `WithPollInterval(d time.Duration) Option`

Sets the interval for checking file changes. Lower values provide faster updates but use more CPU.

```go
tailer.WithPollInterval(500 * time.Millisecond)
```

#### `WithBufferSize(size int) Option`

Sets the channel buffer size. Larger buffers can handle bursts of log lines better.

```go
tailer.WithBufferSize(200)
```

#### `WithLast(n int) Option`

Sets how many lines from the end of the file to read when starting.

```go
tailer.WithLast(20)  // Read last 20 lines on start
```

#### `WithPattern(patterns ...string) Option`

Adds a pattern group for filtering lines. Each pattern is a regular expression. All patterns within a single `WithPattern` call must match (AND logic). Multiple `WithPattern` calls are OR'ed together.

```go
// Match lines containing both "error" AND "database"
tailer.WithPattern("error", "database")

// Match lines with "error" OR "warning"
tailer.New(filepath,
    tailer.WithPattern("error"),
    tailer.WithPattern("warning"),
)
```

#### `WithPlugins(plugins...Plugin) Option`

Adds one or more plugins to process lines before they are sent to the output channel. Plugins can modify line content (e.g., add ANSI color codes) or drop lines entirely. Each plugin's `Apply(line string) (string, bool)` method is called in order - if it returns `false`, the line is dropped and no further plugins are executed.

```go
type Plugin interface {
    // Apply processes a line and returns the modified line
    // and a boolean indicating if processing should continue
    // Return false to drop the line
    Apply(line string) (string, bool)
}
```

#### `WithSyntaxColoring(syntax ...string) Option`

Enable a syntax coloring that adds ANSI color codes to specific patterns in log lines. This is particularly useful for enhancing readability of structured logs in terminal displays.

**Supported syntax styles:**

- **`"level"`, `"levels"`**: Colorizes standard log levels
  - `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`

- **`"slog-text"`**: Colorizes structured logging format (key=value pairs)

- **`"slog-json"`**: Colorizes JSON logging format

**Examples:**

```go
// Colorize log levels only
tail := tailer.New("/var/log/app.log",
    tailer.WithSyntaxColoring("loglevel"),
)

// Colorize both log levels and slog key-value pairs
tail := tailer.New("/var/log/app.log",
    tailer.WithSyntaxColoring("loglevel", "slog"),
)
```

### Terminal Themes

When using the web-based terminal interface via `NewHandler()`, you can customize the terminal appearance using predefined color themes. The terminal uses xterm.js and supports full 16-color ANSI palettes.

#### Available Themes

- **`ThemeDefault`**: Standard dark theme with good contrast
- **`ThemeSolarizedDark`**: Popular Solarized Dark color scheme  
- **`ThemeSolarizedLight`**: Solarized Light for bright environments
- **`ThemeMolokai`**: Vibrant Molokai editor theme
- **`ThemeUbuntu`**: Ubuntu terminal's signature purple theme (default)

#### Terminal Options

The `TerminalOptions` struct allows fine-grained control over terminal behavior and appearance:

```go
type TerminalOptions struct {
    CursorBlink         bool          // Enable/disable cursor blinking
    CursorInactiveStyle string        // Cursor style when terminal is inactive
    CursorStyle         string        // Cursor style: "block", "underline", "bar"
    FontSize            int           // Terminal font size in pixels
    FontFamily          string        // CSS font-family value
    Theme               TerminalTheme // Color theme (see predefined themes)
    Scrollback          int           // Number of lines to keep in scrollback buffer
    DisableStdin        bool          // Disable keyboard input (read-only terminal)
    ConvertEol          bool          // Convert line endings
}
```

**Default terminal settings:**
```go
TerminalOptions{
    CursorBlink:  false,
    FontSize:     12,
    FontFamily:   `"Monaspace Neon", ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace`,
    Theme:        ThemeUbuntu,
    Scrollback:   5000,
    DisableStdin: true, // Terminal is read-only for log viewing
}
```

#### TerminalTheme Structure

Each theme defines a complete 16-color ANSI palette plus UI colors:

```go
type TerminalTheme struct {
    Background          string // Terminal background color
    Foreground          string // Default text color
    Cursor              string // Cursor color
    CursorAccent        string // Cursor accent/border color
    SelectionBackground string // Text selection background
    
    // Standard ANSI colors (0-7)
    Black   string
    Red     string
    Green   string
    Yellow  string
    Blue    string
    Magenta string
    Cyan    string
    White   string
    
    // Bright ANSI colors (8-15)
    BrightBlack   string
    BrightRed     string
    BrightGreen   string
    BrightYellow  string
    BrightBlue    string
    BrightMagenta string
    BrightCyan    string
    BrightWhite   string
}
```

**Example - Solarized Dark theme:**
```go
var ThemeSolarizedDark = TerminalTheme{
    Background:          "#002b36",
    Foreground:          "#839496",
    Cursor:              "#839496",
    CursorAccent:        "#002b36",
    SelectionBackground: "#073642",
    Black:               "#073642",
    Red:                 "#dc322f",
    Green:               "#859900",
    Yellow:              "#b58900",
    Blue:                "#268bd2",
    Magenta:             "#d33682",
    Cyan:                "#2aa198",
    White:               "#eee8d5",
    // ... bright colors
}
```

**Creating custom themes:**

You can define your own themes by creating a `TerminalTheme` with custom colors:

```go
customTheme := tailer.TerminalTheme{
    Background: "#1a1a1a",
    Foreground: "#e0e0e0",
    Cursor:     "#00ff00",
    Red:        "#ff5555",
    Green:      "#50fa7b",
    Yellow:     "#f1fa8c",
    Blue:       "#bd93f9",
    // ... define all colors
}
```

## How It Works

### File Rotation Detection

The tailer detects file rotation by monitoring the file's inode (Unix) or file index (Windows). When a rotation is detected:
1. It reads any remaining content from the old file
2. Opens the new file
3. Continues tailing from the beginning of the new file

### Truncation Detection

The tailer detects file truncation by comparing the current file size with the last known size and read position. When truncation is detected, it seeks to the beginning and reads all new content.

### Server-Sent Events (SSE) Streaming

The HTTP handler uses Server-Sent Events to push log lines to web browsers in real-time:

1. **Connection**: Browser connects to `/watch.stream` endpoint
2. **Streaming**: Server sends each new log line as an SSE `data:` event
3. **Filtering**: Optional `filter` query parameter applies regex patterns server-side
4. **Colorization**: Log levels are automatically wrapped with ANSI color codes for terminal display
5. **Keep-alive**: Regular flush operations keep the connection active
6. **Termination**: Connection closes on browser disconnect, server shutdown, or context cancellation

The SSE format follows the standard:
```
data: <log line with ANSI colors>\n\n
```

### Windows Compatibility

On Windows, files are opened with `FILE_SHARE_DELETE` flag, allowing the file to be renamed or deleted while the tailer has it open. This enables proper log rotation support on Windows.

## Platform Support

- ✅ Windows
- ✅ Linux
- ✅ macOS
- ✅ FreeBSD
- ✅ OpenBSD
- ✅ NetBSD

## Testing

Run the tests:

```bash
go test -v
```

The test suite includes:
- Basic tailing functionality
- Log rotation detection
- File truncation handling
- Pattern filtering

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
