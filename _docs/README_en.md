# SpoofDPI Turkiye

Read in other Languages: [🇹🇷Turkish](https://github.com/renardev/SpoofDPI-Turkiye), [🇬🇧English](https://github.com/renardev/SpoofDPI-Turkiye/blob/main/_docs/README_en.md)

This version of Spoof DPI is configured for use **in Turkiye.**

![image](https://user-images.githubusercontent.com/45588457/148035986-8b0076cc-fefb-48a1-9939-a8d9ab1d6322.png)

## Installation

You can download directly from the [releases](https://github.com/renardev/SpoofDPI-Turkiye/releases) section or
See the installation guide for SpoofDPI [here](https://github.com/renardev/SpoofDPI-Turkiye/blob/main/_docs/INSTALL.md).

## Usage

Since our program is configured specifically for Turkey, you can start and run the version that is right for you directly.

### Advanced Usage

```text
Usage: spoofdpi [options...]
  -addr string
        listen address (default "127.0.0.1")
  -debug
        enable debug output
  -dns-addr string
        dns address (default "77.88.8.8")
  -dns-ipv4-only
        resolve only version 4 addresses
  -dns-port value
        port number for dns (default 1253)
  -enable-doh
        enable 'dns-over-https'
  -pattern value
        bypass DPI only on packets matching this regex pattern; can be given multiple times
  -port value
        port (default 8080)
  -silent
        do not show the banner and server information at start up
  -system-proxy
        enable system-wide proxy (default true)
  -timeout value
        timeout in milliseconds; no timeout when not given
  -v    print spoofdpi's version; this may contain some other relevant information
  -window-size value
        chunk size, in number of bytes, for fragmented client hello,
        try lower values if the default value doesn't bypass the DPI;
        when not given, the client hello packet will be sent in two parts:
        fragmentation for the first data packet and the rest
```

> If you are using any vpn extensions such as Hotspot Shield in Chrome browser,
  go to Settings > Extensions, and disable them.

### OSX

Run `spoofdpi` and it will automatically set your proxy

### Linux

Run `spoofdpi` and open your favorite browser with proxy option

```bash
google-chrome --proxy-server="http://127.0.0.1:8080"
```

## How it works

### HTTP

 Since most websites in the world now support HTTPS, SpoofDPI doesn't bypass Deep Packet Inspections for HTTP requests, However, it still serves proxy connection for all HTTP requests.

### HTTPS

 Although TLS encrypts every handshake process, the domain names are still shown as plaintext in the Client hello packet.
 In other words, when someone else looks on the packet, they can easily guess where the packet is headed to.
 The domain name can offer significant information while DPI is being processed, and we can actually see that the connection is blocked right after sending Client hello packet.
 I had tried some ways to bypass this and found out that it seemed like only the first chunk gets inspected when we send the Client hello packet split into chunks.
 What SpoofDPI does to bypass this is to send the first 1 byte of a request to the server,
 and then send the rest.

## Similar Projects

[GoodbyeDPI-Turkey](https://github.com/cagritaskn/GoodbyeDPI-Turkey) @cagritaskn (Windows)
