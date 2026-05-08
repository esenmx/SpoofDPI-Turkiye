# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Performance / Toolchain
- Bumped Go to `1.26.3` (also reflected in the Dockerfile).
- `cacheKey` builds the hash key with `unsafe.String(unsafe.SliceData(b), len(b))` instead of `string(b)`, eliminating one byte-copy per DNS lookup.
- TLS ClientHello fragmentation now streams via `iter.Seq[[]byte]` (`slices.Chunk`) instead of materializing a `[][]byte` slice. `writeChunks` consumes the iterator directly.
- Per-direction copy buffer in `pipe()` is reused via `sync.Pool` instead of being freshly allocated per accepted connection.
- `ChainResolver` uses `sync.WaitGroup.Go` for cleaner per-resolver goroutine lifecycle.
- `addrselect.commonPrefixLen` uses `math/bits.LeadingZeros8` instead of an open-coded shift loop on bit-mismatch.

### Added
- Parallel-race DNS resolver: tries `-dns-addr`, every `-dns-fallback`, the configured DoH endpoint, and the system resolver in parallel and returns the fastest non-empty answer. Survives single-upstream throttling, hijacking, and RST injection on Turkish ISPs.
- TTL+negative DNS cache that coalesces concurrent in-flight lookups for the same host.
- `-dns-fallback`, `-doh-url`, `-doh-bootstrap-ip` flags.
- TCP-on-truncation retry in the plain DNS path (handles ISPs that return TC=1 for blocked SNIs).
- DoH bootstrap dialer that targets a configured IP so the resolver does not depend on the system resolver to find its own server.
- Continuous-integration workflow (test/lint/build matrix) and golangci-lint config.
- `SECURITY.md`, `CONTRIBUTING.md`, `CODEOWNERS`, `CHANGELOG.md`.
- Per-asset `SHA256SUMS` published with each release.

### Changed
- Default DNS upstream changed from Yandex `77.88.8.8:1253` (which never reached a real server because Yandex serves on `:53`) to Cloudflare `1.1.1.1:53`.
- `-enable-doh` defaults to `true`; default DoH endpoint is `https://cloudflare-dns.com/dns-query` bootstrapped via `1.1.1.1`.
- `-system-proxy` defaults to `true` only on macOS (silently a no-op on Linux/Windows previously).
- Dockerfile and release workflow now build local fork code instead of `github.com/xvzc/SpoofDPI/cmd/spoofdpi@latest`.
- Install script targets `esenmx/SpoofDPI-Turkiye`.
- Per-connection bidirectional copy uses TCP half-close so an EOF in one direction no longer truncates an in-flight response in the other.

### Fixed
- **(hotfix) Forbidden websites no longer load:** the bypass DNS chain raced the system resolver in parallel and returned its answer first, but on Turkish ISPs the system resolver is poisoned. The chunked-ClientHello bypass then dialed the ISP's blocked-page IP and the page never loaded. The bypass chain now contains only the configured upstreams (`-dns-addr`, `-dns-fallback`, DoH); the system resolver is still used for the `useSystemDns=true` (no-pattern-match) path, exactly as before.
- **(hotfix) CI lint job:** golangci-lint v1.62 ships compiled against Go 1.23 and refuses configs targeting Go 1.26.3. Bumped to `v2.6.0` via `golangci/golangci-lint-action@v8` and migrated `.golangci.yaml` to the v2 schema.
- `Client.Exchange` was used instead of `ExchangeContext`, so per-call DNS timeouts were silently ignored. Caller `context.WithTimeout` is now honored.
- `writeChunks` returned `(0, nil)` on Write failures, which left bidirectional goroutines stuck waiting for data that never arrived. The real error is now wrapped and surfaced.
- `net.DialTCP` had no timeout; routes black-holed by ISPs hung tabs for ~75 seconds. Now uses `net.Dialer{Timeout: 10s}.DialContext`.
- `packet.HttpRequest.Tidy()` panicked on requests without a CRLFCRLF body terminator.
- `proxy.Start` accept loop called `logger.Fatal` (which calls `os.Exit` and bypasses the macOS system-proxy unset deferred in `main.go`), leaving the host with a stuck proxy on transient errors. Now retries with backoff and exits cleanly.
- Per-connection goroutine had no panic recovery; one bad request killed the daemon. Added `defer recover()`.
- `regexp.MustCompile` panicked at startup on a typo in `-pattern`. Replaced with `regexp.Compile` and a friendly error message.
- `SystemResolver` ignored `qTypes`, so `-dns-ipv4-only` did not actually filter AAAA.

### Removed
- Dead code in `proxy/{http,https,server}.go` and `proxy/handler/{conn,io}.go`.
- `.github/FUNDING.yml` (routed sponsorships to upstream author).
