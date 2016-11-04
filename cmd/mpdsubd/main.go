// Command mpdsubd provides a Subsonic HTTP API bridge to a backing MPD server.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/fhs/gompd/mpd"
	"github.com/mdlayher/mpdsub"
)

func main() {
	var (
		mpdNetwork  string
		mpdAddr     string
		mpdMusicDir string

		user string
		pass string
		addr string

		verbose bool
	)

	flag.StringVar(&mpdNetwork, "mpd.network", "tcp", "network to use to dial MPD (typically 'tcp' or 'unix')")
	flag.StringVar(&mpdAddr, "mpd.addr", "localhost:6600", "address of MPD server")
	flag.StringVar(&mpdMusicDir, "mpd.music.dir", "", "location of MPD's music directory")

	flag.StringVar(&user, "user", "", "username for authentication to this server")
	flag.StringVar(&pass, "pass", "", "password for authentication to this server")
	flag.StringVar(&addr, "addr", ":4040", "address this server will listen on")

	flag.BoolVar(&verbose, "v", false, "enable verbose logging")

	flag.Parse()

	c, err := mpd.Dial(mpdNetwork, mpdAddr)
	if err != nil {
		log.Fatalf("failed to dial MPD: %v", err)
	}
	log.Printf("connected to MPD: %s://%s", mpdNetwork, mpdAddr)

	s := mpdsub.NewServer(c, &mpdsub.Config{
		SubsonicUser:     user,
		SubsonicPassword: pass,
		MusicDirectory:   mpdMusicDir,
		Verbose:          verbose,
	})

	log.Printf("starting HTTP server: %s", addr)
	if err := http.ListenAndServe(addr, s); err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
}
