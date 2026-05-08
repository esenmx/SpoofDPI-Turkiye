package dns

import (
	"strings"
	"testing"

	"github.com/esenmx/SpoofDPI-Turkiye/util"
)

// TestBypassChainExcludesSystemResolver guards the regression where the
// bypass DNS chain included the local system resolver in its parallel
// race. On Turkish ISPs the system resolver is the closest hop and wins
// the race with the ISP-poisoned IP for blocked domains, so the chunked
// ClientHello bypass dialed the wrong address and forbidden sites stopped
// loading.
//
// The bypass client must contain only the configured upstreams (general
// + DoH) — never `system`.
func TestBypassChainExcludesSystemResolver(t *testing.T) {
	cfg := &util.Config{
		DnsAddr:        "1.1.1.1",
		DnsPort:        53,
		DnsFallback:    []string{"8.8.8.8", "9.9.9.9"},
		EnableDoh:      true,
		DohUrl:         "https://cloudflare-dns.com/dns-query",
		DohBootstrapIp: "1.1.1.1",
	}
	d := NewDns(cfg)

	desc := d.bypassClient.String()
	if strings.Contains(desc, "system") {
		t.Fatalf("bypass chain must not include system resolver, got: %s", desc)
	}
	for _, want := range []string{"general(1.1.1.1:53)", "general(8.8.8.8:53)", "general(9.9.9.9:53)", "doh("} {
		if !strings.Contains(desc, want) {
			t.Errorf("bypass chain missing %q in %s", want, desc)
		}
	}
}

// TestSystemClientStillWiredForOptOutPath ensures the system resolver is
// still available for the useSystemDns=true branch (pattern-not-matched
// requests), which is exactly what the original behavior expected.
func TestSystemClientStillWiredForOptOutPath(t *testing.T) {
	cfg := &util.Config{
		DnsAddr: "1.1.1.1",
		DnsPort: 53,
	}
	d := NewDns(cfg)

	if d.systemClient == nil {
		t.Fatal("system resolver must remain wired for useSystemDns=true branch")
	}
	if !strings.Contains(d.systemClient.String(), "system") {
		t.Fatalf("expected system resolver, got: %s", d.systemClient.String())
	}
}
