# Tuning SpoofDPI for Turkish ISPs

This document collects empirical settings that work on the major Turkish residential and mobile ISPs. Behavior changes over time as ISPs update their middleboxes — please open a PR if your experience differs.

## Defaults

After PR 1–4 the defaults are:

| Flag | Default | Notes |
|------|---------|-------|
| `-dns-addr` | `1.1.1.1` | Cloudflare. Reliable across all major TR ISPs. |
| `-dns-port` | `53` | Standard. Earlier default of `1253` never reached a real server. |
| `-dns-fallback` | `8.8.8.8`, `9.9.9.9` | Tried in parallel with the primary. |
| `-enable-doh` | `true` | DoH on 443 survives port-53 hijacks and Turkcell mobile UDP filtering. |
| `-doh-url` | `https://cloudflare-dns.com/dns-query` | |
| `-doh-bootstrap-ip` | `1.1.1.1` | The DoH client dials this directly so it does not depend on the system resolver. |
| `-window-size` | `5` | Works on most consumer lines. |
| `-system-proxy` | `true` on macOS, `false` elsewhere | |

## ISP Matrix

### Türk Telekom — Fiber

- Default `-window-size 5` works.
- DPI is consistent on the backbone; if a site fails, drop to `-window-size 1`.
- Port 53 is mostly clean for foreign resolvers; Yandex (`77.88.8.8`) is occasionally throttled — Cloudflare/Google are more reliable.

### Türk Telekom — VAE/EAE (DSL)

- Some lines use a more aggressive DPI flavor. Try `-window-size 1` first.
- Port 53 hijack reported on some POPs; `-enable-doh=true` (default) is recommended.

### Turkcell Superonline

- `-window-size 5` usually works.
- Strong DNS hijacking on `8.8.8.8` for blocked SNIs; the fallback chain handles this automatically.
- TCP-on-truncation retry path matters here (PR 2): some POPs return TC=1 for blocked names.

### Vodafone Net

- `-window-size 5` works.
- Selective UDP rate-limiting on resolver IPs has been observed; DoH path stays unaffected.

### Mobile (Turkcell, Vodafone, Türk Telekom)

- Frequent UDP/53 filtering; default DoH path is essential.
- CGNAT can produce quirky NAT-mapping behavior — keep the proxy `-timeout` modest (e.g. `-timeout 60000`) so stale connections recycle.

## Diagnosis Recipe

1. Run with `-debug` and reproduce the failing site.
2. Check the resolution log line:
   - `chain[...]` returning `502 Bad Gateway` — DNS layer issue. Try one resolver at a time:
     `spoofdpi -debug -dns-addr 8.8.8.8 -dns-fallback "" -enable-doh=false`
     vs.
     `spoofdpi -debug -enable-doh=true`
3. If DNS resolves but the page hangs or shows "connection reset", DPI fragmentation isn't winning. Walk `-window-size` down: 5 → 3 → 1 → 0 (legacy single-byte split).
4. If pages load on `127.0.0.1:8080` but the rest of the system doesn't see it, the system proxy didn't get set (Linux/Windows: expected — set the browser proxy manually).

## Reporting

When opening a bug, include:

- ISP and connection type (Fiber, DSL, mobile).
- Output of `spoofdpi -v`.
- Exact command line.
- A `-debug` excerpt covering one failed request from accept to 502/EOF.
