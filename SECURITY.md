# Security Policy

## Reporting a Vulnerability

If you believe you have found a security vulnerability in SpoofDPI-Turkiye, please report it privately by emailing the maintainer (see `git log`) or opening a GitHub Security Advisory at https://github.com/esenmx/SpoofDPI-Turkiye/security/advisories/new.

Please **do not** open a public issue for security-sensitive reports.

Include:

- A description of the issue and its impact.
- Steps to reproduce, or a proof-of-concept.
- Affected version(s) (`spoofdpi -v`).
- Your environment (OS, Go version if building from source, ISP if relevant).

You should expect an acknowledgment within seven days.

## Threat Model

SpoofDPI-Turkiye is a local TLS-fragmentation proxy intended to bypass ISP DPI. It is **not** a security tool and does not protect the confidentiality or integrity of your traffic beyond what TLS already provides. In particular:

- The proxy listens on `127.0.0.1` by default; binding to a non-loopback address will expose it to anyone who can reach that interface.
- Upstream DNS queries and DNS-over-HTTPS responses are validated only insofar as the underlying libraries (Go `net/http`, `miekg/dns`) validate them.
- The TLS ClientHello fragmentation technique is a censorship-circumvention measure, not an anonymity tool.

## Supported Versions

Only the latest released version is supported. Please update before reporting.
