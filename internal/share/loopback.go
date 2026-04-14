package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"photo-tool/internal/config"
)

// Loopback serves Story 3.2–3.3 GET /s/{token} (HTML) and /i/{token} (image) on loopback by default.
type Loopback struct {
	db          *sql.DB
	libraryRoot string
	cfg         config.ShareHTTPConfig

	mu      sync.Mutex
	ln      net.Listener
	srv     *http.Server
	baseURL string
}

// NewLoopback constructs a share server manager. Call EnsureRunning before copying URLs; Close on app shutdown.
// libraryRoot must be the absolute library directory used with the store (Story 3.3 image bytes).
func NewLoopback(db *sql.DB, libraryRoot string, cfg config.ShareHTTPConfig) *Loopback {
	return &Loopback{db: db, libraryRoot: libraryRoot, cfg: cfg}
}

// EnsureRunning starts the HTTP server if needed and returns the clipboard base URL (scheme + host:port, no trailing slash).
func (l *Loopback) EnsureRunning(ctx context.Context) (baseURL string, err error) {
	if l == nil {
		return "", errors.New("share loopback: nil receiver")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ln != nil {
		return l.baseURL, nil
	}

	const maxTries = 10
	var listenErr error
	for offset := 0; offset < maxTries; offset++ {
		port := l.cfg.Port + offset
		if port > 65535 {
			break
		}
		addr := l.cfg.JoinListenAddr(port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			listenErr = err
			if isAddrInUse(err) {
				continue
			}
			return "", fmt.Errorf("share loopback listen %q: %w", addr, err)
		}

		_, effPort, splitErr := net.SplitHostPort(ln.Addr().String())
		if splitErr != nil {
			_ = ln.Close()
			return "", splitErr
		}
		clipHost := l.cfg.ClipboardBaseHost()
		if strings.Contains(clipHost, ":") {
			clipHost = "[" + clipHost + "]"
		}
		base := "http://" + clipHost + ":" + effPort

		srv := &http.Server{
			Handler:           NewHTTPHandler(l.db, l.libraryRoot),
			ReadHeaderTimeout: 10 * time.Second,
		}
		l.ln = ln
		l.srv = srv
		l.baseURL = base
		go func() {
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("share http serve", "err", err)
			}
		}()
		return l.baseURL, nil
	}
	if listenErr == nil {
		listenErr = errors.New("no listen port")
	}
	return "", fmt.Errorf("share loopback: exhausted port tries from %d: %w", l.cfg.Port, listenErr)
}

// Close shuts down the server started by EnsureRunning.
func (l *Loopback) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	srv := l.srv
	l.srv = nil
	l.ln = nil
	l.baseURL = ""
	l.mu.Unlock()
	if srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	// Darwin/Linux "address already in use"; keep string match loose for cross-platform tests.
	s := err.Error()
	return strings.Contains(strings.ToLower(s), "address already in use") ||
		strings.Contains(strings.ToLower(s), "only one usage")
}
