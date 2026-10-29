package llm

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestNewHTTPClientDirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("direct-ok"))
	}))
	defer target.Close()

	client, err := newHTTPClient(ProviderConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("direct client: %v", err)
	}
	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if tr.Proxy != nil {
		t.Fatal("direct transport must not use ProxyFromEnvironment")
	}
	if tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS verification must never be disabled")
	}

	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("direct request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "direct-ok" {
		t.Fatalf("unexpected body %q", body)
	}
}

// TestNewHTTPClientIgnoresAmbientProxyEnv proves the LLM client never honors
// the generic HTTP(S)_PROXY / ALL_PROXY environment variables, so internal
// Docker traffic can never accidentally be routed through a SOCKS proxy.
func TestNewHTTPClientIgnoresAmbientProxyEnv(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "socks5h://127.0.0.1:1")

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	client, err := newHTTPClient(ProviderConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("ambient proxy env must be ignored: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

// --- minimal SOCKS5 (no-auth) server used to prove traffic passes through it ---

func startTestSOCKS5(t *testing.T) (addr string, connects chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("socks listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	connects = make(chan string, 16)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handleTestSOCKSConn(c, connects)
		}
	}()
	return ln.Addr().String(), connects
}

func handleTestSOCKSConn(conn net.Conn, connects chan<- string) {
	defer conn.Close()
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return
	}
	nmethods := int(hdr[1])
	if _, err := io.ReadFull(conn, make([]byte, nmethods)); err != nil {
		return
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}
	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return
	}
	var host string
	switch req[3] {
	case 0x01:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	case 0x03:
		l := make([]byte, 1)
		if _, err := io.ReadFull(conn, l); err != nil {
			return
		}
		dom := make([]byte, int(l[0]))
		if _, err := io.ReadFull(conn, dom); err != nil {
			return
		}
		host = string(dom)
	case 0x04:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	default:
		return
	}
	portB := make([]byte, 2)
	if _, err := io.ReadFull(conn, portB); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portB)
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))
	connects <- target
	// success reply
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	up, err := net.Dial("tcp", target)
	if err != nil {
		return
	}
	defer up.Close()
	go func() { _, _ = io.Copy(up, conn) }()
	_, _ = io.Copy(conn, up)
}

// TestNewHTTPClientSOCKS5 verifies that when LLM_SOCKS5_PROXY is configured the
// HTTP connection really traverses the SOCKS dialer (the local SOCKS server
// records the connect target) while TLS settings remain untouched.
func TestNewHTTPClientSOCKS5(t *testing.T) {
	socksAddr, connects := startTestSOCKS5(t)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("via-socks"))
	}))
	defer target.Close()

	client, err := newHTTPClient(ProviderConfig{Timeout: 5 * time.Second, SOCKS5Proxy: socksAddr})
	if err != nil {
		t.Fatalf("socks client: %v", err)
	}
	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if tr.Proxy != nil {
		t.Fatal("socks transport must not also use ProxyFromEnvironment")
	}
	if tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("TLS verification must never be disabled")
	}

	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatalf("request through socks: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "via-socks" {
		t.Fatalf("unexpected body %q", body)
	}

	// The SOCKS server must have observed the connect to the target address.
	wantHost, wantPort, err := net.SplitHostPort(target.Listener.Addr().String())
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	select {
	case got := <-connects:
		if got != net.JoinHostPort(wantHost, wantPort) {
			t.Fatalf("socks connect target = %q, want %q", got, net.JoinHostPort(wantHost, wantPort))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no SOCKS connect observed — traffic did not pass through the proxy")
	}
}

func TestNewHTTPClientInvalidProxy(t *testing.T) {
	bad := []string{
		"foo",            // missing port
		"host:notaport",  // non-numeric port
		":8080",          // empty host
		"127.0.0.1:",     // empty port
		"http://x:8080",  // scheme not allowed
	}
	for _, addr := range bad {
		_, err := newHTTPClient(ProviderConfig{Timeout: time.Second, SOCKS5Proxy: addr})
		if err == nil {
			t.Errorf("expected config error for invalid proxy %q", addr)
		}
	}
	if _, err := newHTTPClient(ProviderConfig{Timeout: time.Second, SOCKS5Proxy: "127.0.0.1:10808"}); err != nil {
		t.Fatalf("valid proxy rejected: %v", err)
	}
}

// TestSOCKSUnreachableIsRetryable ensures a valid-but-unreachable proxy fails
// at request time with a retryable error (transient), not a permanent one.
func TestSOCKSUnreachableIsRetryable(t *testing.T) {
	// Grab a free port that nothing listens on.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close() // release -> nothing listening

	client, err := newHTTPClient(ProviderConfig{Timeout: time.Second, SOCKS5Proxy: addr})
	if err != nil {
		t.Fatalf("valid-but-closed proxy rejected at construction: %v", err)
	}
	_, err = client.Get("http://127.0.0.1:9")
	if err == nil {
		t.Fatal("expected dial failure")
	}
	// Transport failures surface as ErrProvider with StatusCode 0 -> retryable.
	if !retryable(&ErrProvider{Purpose: PurposeLeadExtraction, Err: err}) {
		t.Fatal("unreachable SOCKS proxy must be classified transient/retryable")
	}
}

func TestValidateSOCKS5Addr(t *testing.T) {
	valid := []string{"127.0.0.1:10808", "host.docker.internal:10808", "[::1]:10808"}
	for _, a := range valid {
		if err := validateSOCKS5Addr(a); err != nil {
			t.Errorf("valid %q rejected: %v", a, err)
		}
	}
	for _, a := range []string{"", "foo", "a:b:c", "host:nan", ":0"} {
		if err := validateSOCKS5Addr(a); err == nil {
			t.Errorf("invalid %q accepted", a)
		}
	}
}

func TestNetworkStage(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&net.DNSError{Name: "x", IsNotFound: true}, "dns"},
		{errors.New("dial tcp: lookup socks5h on 127.0.0.11:53: no such host"), "dns"},
		{errors.New("socks connect tcp 127.0.0.1:10808: connection refused"), "proxy"},
		{errors.New("proxyconnect tcp: dial tcp: connection refused"), "proxy"},
		{errors.New("dial tcp 1.2.3.4:443: connect: connection refused"), "tcp"},
		{errors.New("read tcp: connection reset by peer"), "tcp"},
		{errors.New("tls: handshake failure"), "tls"},
		{errors.New("x509: certificate signed by unknown authority"), "tls"},
		{context.DeadlineExceeded, "timeout"},
		{&net.OpError{Op: "dial", Net: "tcp", Err: context.DeadlineExceeded}, "timeout"},
		{&url.Error{Op: "Get", URL: "https://api.deepseek.com", Err: context.DeadlineExceeded}, "timeout"},
		{errors.New("some http-level error"), "http"},
	}
	for _, c := range cases {
		if got := networkStage(c.err); got != c.want {
			t.Errorf("networkStage(%v) = %q, want %q", c.err, got, c.want)
		}
	}
	if networkStage(nil) != "" {
		t.Fatal("nil error must produce empty stage")
	}
}
