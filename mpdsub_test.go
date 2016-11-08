package mpdsub

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fhs/gompd/mpd"
)

// withServer creates a test Server using the input database and configuration,
// then invokes the input function with the base URL of the server passed as
// a parameter.
func withServer(t *testing.T, db database, fs filesystem, cfg *Config, fn func(base string)) {
	if db == nil {
		db = &memoryDatabase{
			files: []string{},
			attrs: make(map[string]mpd.Attrs, 0),
		}
	}
	if fs == nil {
		fs = &memoryFilesystem{
			files: make(map[string]*memoryFile, 0),
		}
	}

	if cfg == nil {
		cfg = &Config{}
	}
	cfg.Logger = log.New(ioutil.Discard, "", 0)

	s := httptest.NewServer(newServer(db, fs, cfg))
	defer s.Close()

	fn(s.URL)
}

// mustDecodeXML decodes a Subsonic response container from a HTTP response.
func mustDecodeXML(t *testing.T, res *http.Response) container {
	if want, got := http.StatusOK, res.StatusCode; want != got {
		t.Fatalf("unexpected HTTP status code:\n- want: %03d\n-  got: %03d", want, got)
	}

	if want, got := contentTypeXML, res.Header.Get(contentType); want != got {
		t.Fatalf("unexpected response Content-Type:\n- want: %v\n-  got: %v", want, got)
	}

	var c container
	if err := xml.NewDecoder(res.Body).Decode(&c); err != nil {
		t.Fatalf("failed to decode XML: %v", err)
	}
	defer res.Body.Close()

	if want, got := xmlNS, c.XMLNS; want != got {
		t.Fatalf("unexpected XML namespace:\n- want: %v\n-  got: %v", want, got)
	}

	if want, got := apiVersion, c.Version; want != got {
		t.Fatalf("unexpected Subsonic API version:\n- want: %v\n-  got: %v", want, got)
	}

	return c
}

// testRequest performs a single HTTP request against the server specified by base, using the
// input method, target URL, and query parameters.
func testRequest(t *testing.T, base string, method string, target string, values url.Values) *http.Response {
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	u.Path = target
	u.RawQuery = values.Encode()

	r, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		t.Fatalf("failed to create HTTP request: %v", err)
	}

	res, err := (&http.Client{}).Do(r)
	if err != nil {
		t.Fatalf("failed to perform HTTP request: %v", err)
	}

	return res
}

// configAuth returns a Config and url.Values that can be used to authenticate
// successfully in tests.
func configAuth() (*Config, url.Values) {
	const (
		u = "test"
		p = "test"
		c = "test"
		v = "1.14.0"
	)

	cfg := &Config{
		SubsonicUser:     u,
		SubsonicPassword: u,
	}

	values := url.Values{
		"u": []string{u},
		"p": []string{p},
		"c": []string{c},
		"v": []string{v},
	}

	return cfg, values
}

var _ database = &memoryDatabase{}

// A memoryDatabase is an in-memory implementation of database.
type memoryDatabase struct {
	files []string
	attrs map[string]mpd.Attrs
	pingC chan<- struct{}

	mu sync.RWMutex
}

func (db *memoryDatabase) List(args ...string) ([]string, error) {
	if len(args) != 1 || args[0] != "file" {
		panic(fmt.Sprintf("memoryDatabase.List expects argument file, got: %v", args))
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.files, nil
}

func (db *memoryDatabase) Ping() error {
	db.pingC <- struct{}{}
	return nil
}

func (db *memoryDatabase) ReadComments(uri string) (mpd.Attrs, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if attrs, ok := db.attrs[uri]; ok {
		return attrs, nil
	}

	return nil, fmt.Errorf("no MPD attributes for URI: %q", uri)
}

var _ filesystem = &memoryFilesystem{}

// A memoryFilesystem is an in-memory implementation of filesystem.
type memoryFilesystem struct {
	files map[string]*memoryFile

	mu sync.RWMutex
}

func (fs *memoryFilesystem) Open(name string) (file, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if f, ok := fs.files[name]; ok {
		return f, nil
	}

	return nil, os.ErrNotExist
}

// A memoryFile is an in-memory file used by memoryFilesystem.
type memoryFile struct {
	io.ReadSeeker
}

func (f *memoryFile) Close() error               { return nil }
func (f *memoryFile) Stat() (os.FileInfo, error) { return &memoryFileInfo{}, nil }

var _ os.FileInfo = &memoryFileInfo{}

// A memoryFileInfo is an os.FileInfo used by memoryFiles.
type memoryFileInfo struct{}

func (fi *memoryFileInfo) Name() string       { return "" }
func (fi *memoryFileInfo) Size() int64        { return 0 }
func (fi *memoryFileInfo) Mode() os.FileMode  { return 0 }
func (fi *memoryFileInfo) ModTime() time.Time { return time.Now() }
func (fi *memoryFileInfo) IsDir() bool        { return false }
func (fi *memoryFileInfo) Sys() interface{}   { return nil }
