mpdsub [![Build Status](https://travis-ci.org/mdlayher/mpdsub.svg?branch=master)](https://travis-ci.org/mdlayher/mpdsub) [![GoDoc](http://godoc.org/github.com/mdlayher/mpdsub?status.svg)](http://godoc.org/github.com/mdlayher/mpdsub) [![Report Card](https://goreportcard.com/badge/github.com/mdlayher/mpdsub)](https://goreportcard.com/report/github.com/mdlayher/mpdsub)
======

Command `mpdsubd` provides a Subsonic HTTP API bridge to a backing MPD server.

Package `mpdsub` provides types to create a Subsonic HTTP API bridge to a
backing MPD server.

MIT Licensed.

Overview
--------

`mpdsubd` is a service which exposes a Subsonic HTTP API bridge to a backing
MPD server.  MPD manages the music database, and `mpdsubd` queries MPD to
retrieve data to send to Subsonic clients.

`mpdsubd` must have access to the files in MPD's music directory, to enable
streaming to Subsonic clients.  For this reason, it is recommended to run
`mpdsubd` on the same server as MPD.

Usage
-----

Available flags for `mpdsubd` include:

```
$ ./mpdsubd -h
Usage of ./mpdsubd:
  -addr string
        address this server will listen on (default ":4040")
  -mpd.addr string
        address of MPD server (default "localhost:6600")
  -mpd.music.dir string
        location of MPD's music directory
  -mpd.network string
        network to use to dial MPD (typically 'tcp' or 'unix') (default "tcp")
  -pass string
        password for authentication to this server
  -user string
        username for authentication to this server
  -v    enable verbose logging
```

An example of using `mpdsubd` with Subsonic client authentication:

```
$./mpdsubd -mpd.music.dir /var/music -user subsonic -pass mpdsubd
2016/11/04 18:01:59 connected to MPD: tcp://localhost:6600
2016/11/04 18:01:59 starting HTTP server: :4040
```

FAQ
---

At this time, `mpdsubd` is known to work with the following Subsonic clients:

- Subsonic for Android
