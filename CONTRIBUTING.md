# Contributing

Thanks for your interest in SpoofDPI-Turkiye.

## Development Setup

```bash
git clone https://github.com/esenmx/SpoofDPI-Turkiye.git
cd SpoofDPI-Turkiye
go test -race ./...
go build -o /tmp/spoofdpi .
```

Go version is pinned in `go.mod`. CI runs on Linux and macOS.

## Pull Requests

- One logical change per PR. Small, reviewable diffs are preferred.
- Add tests for new behavior. Reproduce-the-bug tests are especially welcome for fixes.
- `go vet ./...` and `go test -race ./...` must pass locally before pushing.
- Conventional Commits style for the subject line is encouraged but not required.
- For changes that affect Turkish ISP behavior, please document which ISP(s) you tested on (Türk Telekom Fiber, Turkcell Superonline, Vodafone Net, mobile carrier, etc.) and which window-size / DNS settings you used.

## Reporting Bugs

Open an issue with:

- `spoofdpi -v` output.
- Your ISP and connection type.
- Reproduction steps with `-debug` enabled.
- Expected vs. observed behavior.

For security issues, see `SECURITY.md`.
