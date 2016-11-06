package mpdsub

import (
	sctx "context"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fhs/gompd/mpd"
)

// A Server is a HTTP server which exposes an emulated Subsonic API in front
// of an MPD server.  It enables Subsonic clients to read information from
// MPD's database and stream files from the local filesystem.
type Server struct {
	db  database
	fs  filesystem
	cfg *Config
	ll  *log.Logger

	mux *http.ServeMux
}

// Config specifies configuration for a Server.
type Config struct {
	// Credentials which Subsonic clients must provide to authenticate
	// to the Server.
	SubsonicUser     string
	SubsonicPassword string

	// MusicDirectory specifies the root music directory for the MPD server.
	// This must match the value specified in MPD's configuration to enable
	// streaming media through the Server.
	//
	// TODO(mdlayher): perhaps enable parsing this via:
	//  - MPD 'config' command, if over UNIX socket
	//  - MPD configuration file
	MusicDirectory string

	// Verbose specifies if the server should enable verbose logging.
	Verbose bool

	// Keepalive specifies an optional duration for how often keepalive messages
	// should be sent to MPD from the Server.  If Keepalive is set to 0,
	// no keepalive messages will be sent to MPD.
	Keepalive time.Duration

	// Logger specifies an optional logger for the Server.  If Logger is
	// nil, Server logs will be sent to stdout.
	Logger *log.Logger
}

// NewServer creates a new Server using the input MPD client and Config.
func NewServer(c *mpd.Client, cfg *Config) *Server {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	}

	return newServer(c, &osFilesystem{}, cfg)
}

// newServer is the internal constructor for Server.  It enables swapping in
// arbitrary database implementations for testing.  It also sets up all Subsonic
// API routes.
func newServer(db database, fs filesystem, cfg *Config) *Server {
	s := &Server{
		db:  db,
		fs:  fs,
		cfg: cfg,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/rest/getLicense.view", s.getLicense)
	mux.HandleFunc("/rest/getIndexes.view", s.getIndexes)
	mux.HandleFunc("/rest/getMusicDirectory.view", s.getMusicDirectory)
	mux.HandleFunc("/rest/getMusicFolders.view", s.getMusicFolders)
	mux.HandleFunc("/rest/ping.view", s.ping)
	mux.HandleFunc("/rest/stream.view", s.stream)

	s.mux = mux

	if cfg.Keepalive > 0 {
		// TODO(mdlayher): enable canceling this goroutine via context or similar
		go s.keepalive(sctx.TODO())
	}

	return s
}

// keepalive sends keepalive messages to the database at regular intervals,
// to keep connections open.
func (s *Server) keepalive(ctx sctx.Context) {
	tick := time.NewTicker(s.cfg.Keepalive)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if err := s.db.Ping(); err != nil {
				s.logf("failed to send keepalive message: %v", err)
			}
		}
	}
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Verbose {
		s.logf("%s -> %s %s", r.RemoteAddr, r.Method, r.URL.String())
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Connection", "close")

	ctx, ok := parseContext(r)
	if !ok {
		// Subsonic API returns HTTP 200 on missing parameters
		writeXML(w, errMissingParameter)
		return
	}

	if ctx.User != s.cfg.SubsonicUser || ctx.Password != s.cfg.SubsonicPassword {
		// Subsonic API returns HTTP 200 on invalid authentication
		writeXML(w, errUnauthorized)
		return
	}

	s.mux.ServeHTTP(w, r)
}

// logf is a convenience function to create a formatted log entry using the
// Server's configured logger.
func (s *Server) logf(format string, v ...interface{}) {
	s.cfg.Logger.Printf(format, v...)
}

// A context is the context for a request, parsed from the HTTP request.
type context struct {
	User     string
	Password string
	Client   string
	Version  string
}

// parseContext parses parameters from a HTTP request into a context.  If any
// mandatory parameters are missing, it returns false.
func parseContext(r *http.Request) (*context, bool) {
	q := r.URL.Query()

	user := q.Get("u")
	if user == "" {
		return nil, false
	}

	// Password may be encoded, so transparently decode it, if needed
	pass := decodePassword(q.Get("p"))
	if pass == "" {
		return nil, false
	}

	client := q.Get("c")
	if client == "" {
		return nil, false
	}

	version := q.Get("v")
	if version == "" {
		return nil, false
	}

	return &context{
		User:     user,
		Password: pass,
		Client:   client,
		Version:  version,
	}, true
}

// decodePassword decodes a password, if necessary, from its encoded hex
// format.  If the password is not encoded, the input string is returned.
func decodePassword(p string) string {
	const prefix = "enc:"

	if !strings.HasPrefix(p, prefix) {
		return p
	}

	// Treat invalid hex as "empty password"
	b, _ := hex.DecodeString(strings.TrimPrefix(p, prefix))
	return string(b)
}
