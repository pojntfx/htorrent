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

## Usage

> TL;DR: Provide a magnet link and path as the `magnet` and `path` URL parameters, authorize using HTTP basic auth or OpenID Connect, and process the resulting HTTP stream using i.e. MPV, cURL or your browser

### 1. Start a Gateway with `htorrent gateway`

The gateway provides the proxy from HTTP to BitTorrent and a info endpoint.

<details>
  <summary>Expand containerized instructions</summary>

```shell
$ sudo mkdir -p /root/.local/share/htorrent/var/lib/htorrent/data/

$ sudo podman run -d --restart=always --label "io.containers.autoupdate=image" --name htorrent-gateway -p 1337:1337 -e API_PASSWORD='myapipassword' -v "/root/.local/share/htorrent/var/lib/htorrent/data/:/root/.local/share/htorrent/var/lib/htorrent/data/" ghcr.io/pojntfx/htorrent htorrent gateway
$ sudo podman generate systemd --new htorrent-gateway | sudo tee /lib/systemd/system/htorrent-gateway.service

$ sudo systemctl daemon-reload

$ sudo systemctl enable --now htorrent-gateway

$ sudo firewall-cmd --permanent --add-port=1337/tcp
$ sudo firewall-cmd --reload
```

</details>

<details>
  <summary>Expand native instructions</summary>

```shell
$ sudo mkdir -p /root/.local/share/htorrent/var/lib/htorrent/data/
$ sudo tee /etc/systemd/system/htorrent-gateway.service<<'EOT'
[Unit]
Description=htorrent Gateway

[Service]
ExecStart=/usr/local/bin/htorrent gateway
Environment="API_PASSWORD=myapipassword"

[Install]
WantedBy=multi-user.target
EOT

$ sudo systemctl daemon-reload

$ sudo systemctl restart htorrent-gateway

$ sudo firewall-cmd --permanent --add-port=1337/tcp
$ sudo firewall-cmd --reload
```

</details>

It should now be reachable on [localhost:1337](http://localhost:1337/).

To use it in production, put this gateway behind a TLS-enabled reverse proxy such as [Caddy](https://caddyserver.com/) or [Traefik](https://traefik.io/). For the best security, you should use OpenID Connect to authenticate; for more information, see the [gateway reference](#gateway). You can also embed the gateway in your own application using it's [Go API](https://pkg.go.dev/github.com/pojntfx/htorrent/pkg/server).

### 2. Get Torrent Infos with `htorrent info`

First, set the remote address:

```shell
$ export RADDR='http://localhost:1337/'
```

Next, set the API password using the `API_PASSWORD` env variable:

```shell
$ export API_PASSWORD='myapipassword'
```

If you use OIDC to authenticate, you can instead set the API password using [goit](https://github.com/pojntfx/goit) like so:

```shell
$ export OIDC_CLIENT_ID='Ab7OLrQibhXUzKHGWYDFieLa2KqZmFzb' OIDC_ISSUER='https://pojntfx.eu.auth0.com/' OIDC_REDIRECT_URL='http://localhost:11337'
$ export API_PASSWORD="$(goit)"
```

If you want to now get information on a torrent, you can search it by magnet link:

```shell
$ htorrent info -m='magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent'
name: Sintel
infohash: 08ada5a7a6183aae1e09d831df6748d566095a10
description: ""
creationDate: 1659737923
files:
    - path: Sintel/Sintel.de.srt
      length: 1652
      streamURL: http://localhost:1337/stream?magnet=magnet%3A%3Fxt%3Durn%3Abtih%3A08ada5a7a6183aae1e09d831df6748d566095a10%26dn%3DSintel%26tr%3Dudp%253A%252F%252Fexplodie.org%253A6969%26tr%3Dudp%253A%252F%252Ftracker.coppersurfer.tk%253A6969%26tr%3Dudp%253A%252F%252Ftracker.empire-js.us%253A1337%26tr%3Dudp%253A%252F%252Ftracker.leechers-paradise.org%253A6969%26tr%3Dudp%253A%252F%252Ftracker.opentrackr.org%253A1337%26tr%3Dwss%253A%252F%252Ftracker.btorrent.xyz%26tr%3Dwss%253A%252F%252Ftracker.fastcast.nz%26tr%3Dwss%253A%252F%252Ftracker.openwebtorrent.com%26ws%3Dhttps%253A%252F%252Fwebtorrent.io%252Ftorrents%252F%26xs%3Dhttps%253A%252F%252Fwebtorrent.io%252Ftorrents%252Fsintel.torrent&path=Sintel%2FSintel.de.srt
    - path: Sintel/Sintel.en.srt
      length: 1514
      streamURL: http://localhost:1337/stream?magnet=magnet%3A%3Fxt%3Durn%3Abtih%3A08ada5a7a6183aae1e09d831df6748d566095a10%26dn%3DSintel%26tr%3Dudp%253A%252F%252Fexplodie.org%253A6969%26tr%3Dudp%253A%252F%252Ftracker.coppersurfer.tk%253A6969%26tr%3Dudp%253A%252F%252Ftracker.empire-js.us%253A1337%26tr%3Dudp%253A%252F%252Ftracker.leechers-paradise.org%253A6969%26tr%3Dudp%253A%252F%252Ftracker.opentrackr.org%253A1337%26tr%3Dwss%253A%252F%252Ftracker.btorrent.xyz%26tr%3Dwss%253A%252F%252Ftracker.fastcast.nz%26tr%3Dwss%253A%252F%252Ftracker.openwebtorrent.com%26ws%3Dhttps%253A%252F%252Fwebtorrent.io%252Ftorrents%252F%26xs%3Dhttps%253A%252F%252Fwebtorrent.io%252Ftorrents%252Fsintel.torrent&path=Sintel%2FSintel.en.srt
# ...
```

Alternatively, you can also do this using cURL directly:

```shell
$ curl -u "admin:${API_PASSWORD}" -L -G --data-urlencode 'magnet=magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent' http://localhost:1337/info
[{"path":"Sintel/Sintel.de.srt","length":1652,"creationTime":1655501577},{"path":"Sintel/Sintel.en.srt","length":1514,"creationTime":1655501577},{"path":"Sintel/Sintel.es.srt","length":1554,"creationTime":1655501577},{"path":"Sintel/Sintel.fr.srt","length":1618,"creationTime":1655501577},{"path":"Sintel/Sintel.it.srt","length":1546,"creationTime":1655501577},{"path":"Sintel/Sintel.mp4","length":129241752,"creationTime":1655501577},{"path":"Sintel/Sintel.nl.srt","length":1537,"creationTime":1655501577},{"path":"Sintel/Sintel.pl.srt","length":1536,"creationTime":1655501577},{"path":"Sintel/Sintel.pt.srt","length":1551,"creationTime":1655501577},{"path":"Sintel/Sintel.ru.srt","length":2016,"creationTime":1655501577},{"path":"Sintel/poster.jpg","length":46115,"creationTime":1655501577}
```

For more information, see the [info reference](#info). You can also embed the client in your own application using it's [Go API](https://pkg.go.dev/github.com/pojntfx/htorrent/pkg/client).

### 3. Stream using a Media Player or cURL with `htorrent info -x`

If you want to stream the file directly by selecting i.e. the first `.mkv` or `.mp4` file in the torrent, you can use the `-x` flag to return the URL directly:

```shell
$ htorrent info -m='magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent' -x='(.*).mp4'
http://localhost:1337/stream?magnet=magnet%3A%3Fxt%3Durn%3Abtih%3A08ada5a7a6183aae1e09d831df6748d566095a10%26dn%3DSintel%26tr%3Dudp%253A%252F%252Fexplodie.org%253A6969%26tr%3Dudp%253A%252F%252Ftracker.coppersurfer.tk%253A6969%26tr%3Dudp%253A%252F%252Ftracker.empire-js.us%253A1337%26tr%3Dudp%253A%252F%252Ftracker.leechers-paradise.org%253A6969%26tr%3Dudp%253A%252F%252Ftracker.opentrackr.org%253A1337%26tr%3Dwss%253A%252F%252Ftracker.btorrent.xyz%26tr%3Dwss%253A%252F%252Ftracker.fastcast.nz%26tr%3Dwss%253A%252F%252Ftracker.openwebtorrent.com%26ws%3Dhttps%253A%252F%252Fwebtorrent.io%252Ftorrents%252F%26xs%3Dhttps%253A%252F%252Fwebtorrent.io%252Ftorrents%252Fsintel.torrent&path=Sintel%2FSintel.mp4
```

If you want to stream the resulting file in a video player like [MPV](https://mpv.io/), run it like so (note `--http-header-fields` for authentication):

```shell
$ mpv "$(htorrent info -m='magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent' -x='(.*).mp4')" --http-header-fields="Authorization: Basic $(printf admin:${API_PASSWORD} | base64 -w0)"
```

Alternatively, you can also download the stream by using cURL directly:

```shell
$ curl -u "admin:${API_PASSWORD}" -L -G --data-urlencode 'magnet=magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent' --data-urlencode 'path=Sintel/Sintel.mp4' http://localhost:1337/stream -o ~/Downloads/sintel.mp4
# ...
$ file ~/Downloads/sintel.mp4
/home/pojntfx/Downloads/sintel.mp4: ISO Media, MP4 Base Media v1 [ISO 14496-12:2003]
```

For more information, see the [info reference](#info).

#### 4. Get Torrent Metrics with `htorrent metrics`

If you want to check metrics such as download progress or the amount of connected peers, you can use the metrics endpoint:

```shell
$ htorrent metrics
- magnet: magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F
  infohash: 08ada5a7a6183aae1e09d831df6748d566095a10
  peers: 2
  files:
    - path: Sintel/Sintel.de.srt
      length: 1652
      completed: 1652
    - path: Sintel/Sintel.en.srt
      length: 1514
      completed: 1514
    - path: Sintel/Sintel.es.srt
```

For more information, see the [metrics reference](#metrics).

🚀 **That's it!** We hope you enjoy using hTorrent.

## Reference

### Command Line Arguments

```shell
$ htorrent --help
Access torrents just like you would access any file served over a web server!


Find more information at:
https://github.com/pojntfx/htorrent

Usage:
  htorrent [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  gateway     Start a gateway
  help        Help about any command
  info        Get streamable URLs from the gateway's info endpoint

Flags:
  -h, --help          help for htorrent
  -v, --verbose int   Verbosity level (0 is disabled, default is info, 7 is trace) (default 5)

Use "htorrent [command] --help" for more information about a command.
```

<details>
  <summary>Expand subcommand reference</summary>

#### Gateway

```shell
$ htorrent gateway --help
Start a gateway

Usage:
  htorrent gateway [flags]

Aliases:
  gateway, g

Flags:
      --api-password string     Password for the management API (can also be set using the API_PASSWORD env variable). Ignored if any of the OIDC parameters are set.
      --api-username string     Username for the management API (can also be set using the API_USERNAME env variable). Ignored if any of the OIDC parameters are set. (default "admin")
  -h, --help                    help for gateway
  -l, --laddr string            Listening address (default ":1337")
      --oidc-client-id string   OIDC Client ID (i.e. myoidcclientid) (can also be set using the OIDC_CLIENT_ID env variable)
      --oidc-issuer string      OIDC Issuer (i.e. https://pojntfx.eu.auth0.com/) (can also be set using the OIDC_ISSUER env variable)
  -s, --storage string          Path to store downloaded torrents in (default "/home/pojntfx/.local/share/htorrent/var/lib/htorrent/data")

Global Flags:
  -v, --verbose int   Verbosity level (0 is disabled, default is info, 7 is trace) (default 5)
```

#### Info

```shell
$ htorrent info --help
Get streamable URLs and other info for a magnet link from the gateway

Usage:
  htorrent info [flags]

Aliases:
  info, i

Flags:
  -p, --api-password string   Username or OIDC access token for the gateway
  -u, --api-username string   Username for the gateway (default "admin")
  -x, --expression string     Regex to select the link to output by, i.e. (.*).mkv$ to only return the first .mkv file; disables all other info
  -h, --help                  help for info
  -m, --magnet string         Magnet link to get info for
  -r, --raddr string          Remote address (default "http://localhost:1337/")

Global Flags:
  -v, --verbose int   Verbosity level (0 is disabled, default is info, 7 is trace) (default 5)
```

#### Metrics

```shell
$ htorrent metrics --help
Get metrics from the gateway

Usage:
  htorrent metrics [flags]

Aliases:
  metrics, m

Flags:
  -p, --api-password string   Username or OIDC access token for the gateway
  -u, --api-username string   Username for the gateway (default "admin")
  -h, --help                  help for metrics
  -r, --raddr string          Remote address (default "http://localhost:1337/")

Global Flags:
  -v, --verbose int   Verbosity level (0 is disabled, default is info, 7 is trace) (default 5)
```

</details>

### Environment Variables

All command line arguments described above can also be set using environment variables; for example, to set `--raddr` to `http://example.com:443/` with an environment variable, use `RADDR=http://example.com:443/`.

## Acknowledgements

- [anacrolix/torrent](https://github.com/anacrolix/torrent) provides the BitTorrent library.

To all the rest of the authors who worked on the dependencies used: **Thanks a lot!**

## Contributing

To contribute, please use the [GitHub flow](https://guides.github.com/introduction/flow/) and follow our [Code of Conduct](./CODE_OF_CONDUCT.md).

To build and start a development version of hTorrent locally, run the following:

```shell
$ git clone https://github.com/pojntfx/htorrent.git
$ cd htorrent
$ make depend
$ make && sudo make install
$ export API_PASSWORD='myapipassword'
$ htorrent gateway # Starts the gateway
# In another terminal
$ export API_PASSWORD='myapipassword'
$ htorrent info -m='magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent' # Get info
$ mpv "$(htorrent info -m='magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent' -x='(.*).mp4')" --http-header-fields="Authorization: Basic $(printf admin:${API_PASSWORD} | base64 -w0)" # Stream using MPV
```

Have any questions or need help? Chat with us [on Matrix](https://matrix.to/#/#htorrent:matrix.org?via=matrix.org)!

## License

hTorrent (c) 2022 Felicitas Pojtinger and contributors

SPDX-License-Identifier: AGPL-3.0
