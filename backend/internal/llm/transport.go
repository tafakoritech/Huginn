package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// Transport mode labels for startup logging.
const (
	transportModeDirect = "direct"
	transportModeSOCKS5 = "socks5"
)

// newHTTPClient builds the LLM HTTP client with an explicit transport.
//
//   - Direct mode (SOCKS5Proxy empty): a plain transport that dials the
//     provider directly and never consults HTTP(S)_PROXY/ALL_PROXY.
//   - SOCKS5 mode (SOCKS5Proxy set): the transport's DialContext delegates to a
//     SOCKS5 dialer pointed at cfg.SOCKS5Proxy. Only the TCP connection goes
//     through the proxy; TLS still terminates against the real provider host
//     (certificate verification is never disabled).
//
// A malformed proxy address is a configuration error and is returned so the
// caller can fail fast instead of retrying forever at runtime.
func newHTTPClient(cfg ProviderConfig) (*http.Client, error) {
	// Direct is a concrete type; box it as the Dialer interface so both the
	// direct and SOCKS5 dialers share one code path below.
	var dialer proxy.Dialer = proxy.Direct

	if cfg.SOCKS5Proxy != "" {
		if err := validateSOCKS5Addr(cfg.SOCKS5Proxy); err != nil {
			return nil, err
		}
		sd, err := proxy.SOCKS5("tcp", cfg.SOCKS5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("llm: build socks5 dialer for %q: %w", cfg.SOCKS5Proxy, err)
		}
		dialer = sd
	}

	transport := &http.Transport{
		// Explicitly ignore ambient HTTP(S)_PROXY/ALL_PROXY: only the dedicated
		// LLM_SOCKS5_PROXY (applied above) may route LLM traffic.
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if cd, ok := dialer.(proxy.ContextDialer); ok {
				return cd.DialContext(ctx, network, addr)
			}
			return dialer.Dial(network, addr)
		},
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}

	// No client-level Timeout here: the provider applies a fresh, full
	// per-attempt context.WithTimeout (see attemptContext) for every network
	// attempt, so the configured request timeout is never shared/consumed
	// across retries.
	return &http.Client{
		Transport: transport,
	}, nil
}

// LogTransportConfig emits a startup log line describing which LLM transport
// mode is active. It never logs the API key or the Authorization header.
func LogTransportConfig(l *slog.Logger, cfg ProviderConfig) {
	mode := transportModeDirect
	proxy := ""
	if cfg.SOCKS5Proxy != "" {
		mode = transportModeSOCKS5
		proxy = cfg.SOCKS5Proxy
	}
	l.Info("llm transport configured",
		"mode", mode,
		"proxy", proxy,
		"provider", cfg.BaseURL,
		"model", cfg.Model,
	)
}

// validateSOCKS5Addr rejects malformed "host:port" proxy addresses so proxy
// configuration mistakes surface as startup errors, not infinite retries.
func validateSOCKS5Addr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("llm: invalid LLM_SOCKS5_PROXY %q (want host:port): %w", addr, err)
	}
	if host == "" {
		return fmt.Errorf("llm: invalid LLM_SOCKS5_PROXY %q: empty host", addr)
	}
	if port == "" {
		return fmt.Errorf("llm: invalid LLM_SOCKS5_PROXY %q: empty port", addr)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("llm: invalid LLM_SOCKS5_PROXY %q: port %q is not numeric", addr, port)
	}
	return nil
}

// networkStage classifies a transport error so operators can tell whether an
// LLM request failed during DNS, the proxy connection, TCP, TLS, the HTTP
// request itself, or a timeout.
func networkStage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no such host"):
		return "dns"
	case strings.Contains(msg, "server misbehaving"):
		return "dns"
	case strings.Contains(msg, "proxyconnect"):
		return "proxy"
	case strings.Contains(msg, "socks connect"):
		return "proxy"
	case strings.Contains(msg, "connection refused"):
		return "tcp"
	case strings.Contains(msg, "connection reset"):
		return "tcp"
	case strings.Contains(msg, "eof"):
		return "tcp"
	case strings.Contains(msg, "certificate"), strings.Contains(msg, "x509"),
		strings.Contains(msg, "tls:"):
		return "tls"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline exceeded"):
		return "timeout"
	case strings.Contains(msg, "lookup"):
		return "dns"
	default:
		return "http"
	}
}
