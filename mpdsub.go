// Package mpdsub provides types to create a Subsonic HTTP API bridge to
// a backing MPD server.
package mpdsub

import (
	"io"
	"os"

	"github.com/fhs/gompd/mpd"
)

var _ database = &mpd.Client{}

// A database is a type which can return data in the same format as MPD
// database queries.  database is implemented by *mpd.Client.
type database interface {
	List(args ...string) ([]string, error)
	ReadComments(uri string) (mpd.Attrs, error)
}

// A filesystem is a type which can open a file.  filesystem is implemented
// by *osFilesystem.
type filesystem interface {
	Open(name string) (file, error)
}

var _ filesystem = &osFilesystem{}

// An osFilesystem is a small wrapper around package os, which implements
// filesystem.
type osFilesystem struct{}

// Open opens a file in the filesystem using os.Open.
func (*osFilesystem) Open(name string) (file, error) {
	return os.Open(name)
}

var _ file = &os.File{}

// A file is a type which can be opened using a filesystem.  file is implemented
// by *os.File.
type file interface {
	Stat() (os.FileInfo, error)
	io.Closer
	io.ReadSeeker
}
