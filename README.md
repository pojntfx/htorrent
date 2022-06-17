# hTorrent

![Logo](./docs/logo-readme.png)

HTTP to BitTorrent gateway with seeking support. Access torrents just like you would access any file served over a web server!

[![hydrun CI](https://github.com/pojntfx/htorrent/actions/workflows/hydrun.yaml/badge.svg)](https://github.com/pojntfx/htorrent/actions/workflows/hydrun.yaml)
[![Docker CI](https://github.com/pojntfx/htorrent/actions/workflows/docker.yaml/badge.svg)](https://github.com/pojntfx/htorrent/actions/workflows/docker.yaml)
![Go Version](https://img.shields.io/badge/go%20version-%3E=1.18-61CFDD.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/pojntfx/htorrent.svg)](https://pkg.go.dev/github.com/pojntfx/htorrent)
[![Matrix](https://img.shields.io/matrix/htorrent:matrix.org)](https://matrix.to/#/#htorrent:matrix.org?via=matrix.org)
[![Binary Downloads](https://img.shields.io/github/downloads/pojntfx/htorrent/total?label=binary%20downloads)](https://github.com/pojntfx/htorrent/releases)

## Overview

hTorrent is a gateway that allows you to access torrents over HTTP.

It enables you to ...

- **Stream torrents with (almost) any video player**: By setting the gateway as an HTTP source for a media player, you can stream & seek any torrent without downloading it.
- **Download torrents without a torrent client**: Using i.e. cURL or a web browser, you can download a torrent just like you would download any other file.
- **Build web apps that consume torrent sources**: While [WebTorrent](https://webtorrent.io/) is a way to stream torrents via WebRTC, hTorrent can provide a gateway for clients that don't support that capability.

## Installation

### Containerized

You can get the OCI image like so:

```shell
$ podman pull ghcr.io/pojntfx/htorrent
```

### Natively

Static binaries are available on [GitHub releases](https://github.com/pojntfx/htorrent/releases).

On Linux, you can install them like so:

```shell
$ curl -L -o /tmp/htorrent "https://github.com/pojntfx/htorrent/releases/latest/download/htorrent.linux-$(uname -m)"
$ sudo install /tmp/htorrent /usr/local/bin
```

On macOS, you can use the following:

```shell
$ curl -L -o /tmp/htorrent "https://github.com/pojntfx/htorrent/releases/latest/download/htorrent.darwin-$(uname -m)"
$ sudo install /tmp/htorrent /usr/local/bin
```

On Windows, the following should work (using PowerShell as administrator):

```shell
PS> Invoke-WebRequest https://github.com/pojntfx/htorrent/releases/latest/download/htorrent.windows-x86_64.exe -OutFile \Windows\System32\htorrent.exe
```

You can find binaries for more operating systems and architectures on [GitHub releases](https://github.com/pojntfx/htorrent/releases).

## License

hTorrent (c) 2022 Felicitas Pojtinger and contributors

SPDX-License-Identifier: AGPL-3.0
