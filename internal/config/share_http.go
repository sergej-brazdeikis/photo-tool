package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	// EnvShareHTTPHost is the listen host for the in-process share HTTP server (default loopback literal).
	EnvShareHTTPHost = "PHOTO_TOOL_SHARE_HTTP_HOST"
	// EnvShareHTTPPort is the preferred TCP port (decimal). If the port is busy, Loopback may try a short run of successors.
	EnvShareHTTPPort = "PHOTO_TOOL_SHARE_HTTP_PORT"
	// EnvShareHTTPBindAll, when set to "1" or "true" (case-insensitive), binds 0.0.0.0 instead of the configured host.
	// LAN-wide exposure must stay explicit; see package comment on ShareHTTPConfig.
	EnvShareHTTPBindAll = "PHOTO_TOOL_SHARE_HTTP_BIND_ALL"
)

// DefaultShareHTTPPort is the first port tried when PHOTO_TOOL_SHARE_HTTP_PORT is unset.
const DefaultShareHTTPPort = 8765

// ShareHTTPConfig holds loopback-first share server settings (Story 3.2 / NFR-06).
type ShareHTTPConfig struct {
	// Host is the interface address passed to net.Listen (default "127.0.0.1").
	Host string
	// Port is the first candidate port; may advance on EADDRINUSE.
	Port int
	// BindAll requests 0.0.0.0 instead of Host — explicit LAN exposure; never the default.
	BindAll bool
}

// LoadShareHTTPConfig reads env overrides. Unset port uses DefaultShareHTTPPort.
func LoadShareHTTPConfig() (ShareHTTPConfig, error) {
	host := strings.TrimSpace(os.Getenv(EnvShareHTTPHost))
	if host == "" {
		host = "127.0.0.1"
	}
	port := DefaultShareHTTPPort
	if v := strings.TrimSpace(os.Getenv(EnvShareHTTPPort)); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			return ShareHTTPConfig{}, fmt.Errorf("%s: invalid port %q", EnvShareHTTPPort, v)
		}
		port = p
	}
	bindAll := parseEnvBool(os.Getenv(EnvShareHTTPBindAll))
	return ShareHTTPConfig{Host: host, Port: port, BindAll: bindAll}, nil
}

// ListenHost returns the host string for net.Listen("tcp", ...).
func (c ShareHTTPConfig) ListenHost() string {
	if c.BindAll {
		// SECURITY: binds all interfaces — only when PHOTO_TOOL_SHARE_HTTP_BIND_ALL is set.
		return "0.0.0.0"
	}
	return c.Host
}

// ClipboardBaseHost returns the host segment for user-facing URLs copied to the clipboard.
// When BindAll is false this matches ListenHost. When BindAll is true we still prefer a loopback
// literal for “open on this machine” URLs; document that remote clients need the machine LAN IP.
func (c ShareHTTPConfig) ClipboardBaseHost() string {
	if c.BindAll {
		return "127.0.0.1"
	}
	return c.Host
}

// JoinListenAddr builds host:port for Listen.
func (c ShareHTTPConfig) JoinListenAddr(port int) string {
	h := c.ListenHost()
	if strings.Contains(h, ":") {
		return net.JoinHostPort(h, strconv.Itoa(port))
	}
	return fmt.Sprintf("%s:%d", h, port)
}

func parseEnvBool(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	return s == "1" || s == "true" || s == "yes"
}
