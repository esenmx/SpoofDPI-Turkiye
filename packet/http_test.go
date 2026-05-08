package packet

import (
	"strings"
	"testing"
)

func TestParseAndTidyDropsProxyConnection(t *testing.T) {
	raw := "GET http://example.com/ HTTP/1.1\r\nHost: example.com\r\nProxy-Connection: keep-alive\r\nUser-Agent: t\r\n\r\n"
	p, err := ReadHttpRequest(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	p.Tidy()
	out := string(p.Raw())
	if strings.Contains(out, "Proxy-Connection") {
		t.Fatalf("Proxy-Connection should be stripped, got:\n%s", out)
	}
	if !strings.Contains(out, "GET / HTTP/1.1") {
		t.Fatalf("expected request-line rewritten with path-only, got:\n%s", out)
	}
}

// TestTidyDoesNotPanicOnMissingBodyTerminator guards the original
// `parts[1]` index-out-of-range panic on requests without the
// trailing CRLFCRLF.
func TestTidyDoesNotPanicOnMissingBodyTerminator(t *testing.T) {
	p := &HttpRequest{
		raw:     []byte("CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n"),
		method:  "CONNECT",
		domain:  "example.com",
		port:    "443",
		path:    "",
		version: "HTTP/1.1",
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Tidy panicked: %v", r)
		}
	}()
	p.Tidy()
	if len(p.Raw()) == 0 {
		t.Fatal("expected non-empty output")
	}
}

func TestIsValidMethod(t *testing.T) {
	for _, m := range []string{"GET", "CONNECT", "PUT", "PATCH"} {
		p := &HttpRequest{method: m}
		if !p.IsValidMethod() {
			t.Errorf("%s should be valid", m)
		}
	}
	p := &HttpRequest{method: "BOGUS"}
	if p.IsValidMethod() {
		t.Error("BOGUS should be rejected")
	}
}
