package mpdsub

import (
	"encoding/xml"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fhs/gompd/mpd"
)

func TestServer_getIndexes(t *testing.T) {
	tests := []struct {
		name string
		db   database

		indexes []index
	}{
		{
			name: "one MP3",
			db: &memoryDatabase{
				files: []string{"A.mp3"},
			},
			indexes: []index{{
				Name: "A",
				Artists: []artist{{
					Name: "A.mp3",
					ID:   "0",
				}},
			}},
		},
		{
			name: "one MP3 and one artist folder",
			db: &memoryDatabase{
				files: []string{
					"A.mp3",
					"B/B.mp3",
				},
			},
			indexes: []index{
				{
					Name: "A",
					Artists: []artist{{
						Name: "A.mp3",
						ID:   "0",
					}},
				},
				{
					Name: "B",
					Artists: []artist{{
						Name: "B",
						ID:   "1",
					}},
				},
			},
		},
		{
			name: "three artists, two with same initial letter",
			db: &memoryDatabase{
				files: []string{
					"Apple/A.mp3",
					"Banana/B.mp3",
					"Blueberry/B.mp3",
				},
			},
			indexes: []index{
				{
					Name: "A",
					Artists: []artist{{
						Name: "Apple",
						ID:   "0",
					}},
				},
				{
					Name: "B",
					Artists: []artist{
						{
							Name: "Banana",
							ID:   "2",
						},
						{
							Name: "Blueberry",
							ID:   "4",
						},
					},
				},
			},
		},
		{
			name: "multiple artists, two with beginning digits",
			db: &memoryDatabase{
				files: []string{
					"123/abc/abc.mp3",
					"456/def/def.mp3",
					"Apple/A.mp3",
					"Banana/B.mp3",
					"Blueberry/B.mp3",
				},
			},
			indexes: []index{
				{
					Name: "#",
					Artists: []artist{
						{
							Name: "123",
							ID:   "0",
						},
						{
							Name: "456",
							ID:   "3",
						},
					},
				},
				{
					Name: "A",
					Artists: []artist{{
						Name: "Apple",
						ID:   "6",
					}},
				},
				{
					Name: "B",
					Artists: []artist{
						{
							Name: "Banana",
							ID:   "8",
						},
						{
							Name: "Blueberry",
							ID:   "10",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, values := configAuth()
			withServer(t, tt.db, nil, cfg, func(base string) {
				c := mustDecodeXML(t, testRequest(t, base, http.MethodGet, "/rest/getIndexes.view", values))

				if c.Indexes == nil {
					t.Fatal("indexes is nil")
				}

				mustIndexesEqual(t, tt.indexes, c.Indexes.Indexes)
			})
		})
	}
}

func TestServer_getLicense(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, values := configAuth()
			withServer(t, nil, nil, cfg, func(base string) {
				c := mustDecodeXML(t, testRequest(t, base, http.MethodGet, "/rest/getLicense.view", values))

				if c.License == nil {
					t.Fatal("license is nil")
				}

				if want, got := true, c.License.Valid; want != got {
					t.Fatalf("unexpected license valid value:\n- want: %v\n-  got: %v", want, got)
				}
			})
		})
	}
}

func TestServer_getMusicDirectory(t *testing.T) {
	tests := []struct {
		name string
		db   database

		id string

		xmlError *subsonicError
		httpCode int
		mdc      *musicDirectoryContainer
	}{
		{
			name: "no ID",

			xmlError: &subsonicError{Code: codeMissingParameter},
		},
		{
			name: "bad ID",

			id: "foo",

			xmlError: &subsonicError{Code: codeGeneric},
		},
		{
			name: "no files",

			id: "0",

			httpCode: http.StatusNotFound,
		},
		{
			name: "one file",
			db: &memoryDatabase{
				files: []string{
					"foo/foo.mp3",
					"foo/bar.mp3",
					"foo/bar/baz.mp3",
					"bar/bar.mp3",
				},
				attrs: map[string]mpd.Attrs{
					"foo/foo.mp3": mpd.Attrs{
						"TITLE": "foo",
					},
					"foo/bar.mp3": mpd.Attrs{
						"TITLE": "bar",
					},
				},
			},

			id: "0",

			mdc: &musicDirectoryContainer{
				ID:   "0",
				Name: "foo/foo.mp3",

				Children: []child{
					{
						ID:    "1",
						Title: "foo",
					},
					{
						ID:    "2",
						Title: "bar",
					},
					{
						ID:    "3",
						Title: "bar",
						IsDir: true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, values := configAuth()

			if tt.id != "" {
				values.Set("id", tt.id)
			}

			withServer(t, tt.db, nil, cfg, func(base string) {
				res := testRequest(t, base, http.MethodGet, "/rest/getMusicDirectory.view", values)

				if tt.httpCode != 0 {
					if want, got := tt.httpCode, res.StatusCode; want != got {
						t.Fatalf("unexpected HTTP status code:\n- want: %03d\n-  got: %03d",
							want, got)
					}

					return
				}

				c := mustDecodeXML(t, res)

				if tt.xmlError != nil {
					if want, got := tt.xmlError.Code, c.Error.Code; want != got {
						t.Fatalf("unexpected XML error code::\n- want: %v\n-  got: %v",
							want, got)
					}

					return
				}

				mustMusicDirectoryContainersEqual(t, tt.mdc, c.MusicDirectory)
			})
		})
	}
}

func TestServer_getMusicFolders(t *testing.T) {
	tests := []struct {
		name   string
		folder string
	}{
		{
			name:   "music",
			folder: "/var/music",
		},
		{
			name:   "FLAC",
			folder: "/srv/media/Music/FLAC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, values := configAuth()
			cfg.MusicDirectory = tt.folder

			withServer(t, nil, nil, cfg, func(base string) {
				c := mustDecodeXML(t, testRequest(t, base, http.MethodGet, "/rest/getMusicFolders.view", values))

				if c.MusicFolders == nil {
					t.Fatal("music folders is nil")
				}

				if want, got := 1, len(c.MusicFolders.MusicFolders); want != got {
					t.Fatalf("unexpected number of music folders:\n- want: %v\n-  got: %v", want, got)
				}

				if want, got := tt.name, c.MusicFolders.MusicFolders[0].Name; want != got {
					t.Fatalf("unexpected music folder name:\n- want: %q\n-  got: %q", want, got)
				}
			})
		})
	}
}

func TestServer_stream(t *testing.T) {
	const (
		musicDirectory = "/var/music"

		audioFLAC = "audio/flac"
		audioMPEG = "audio/mpeg"
	)

	tests := []struct {
		name string
		db   database
		fs   filesystem

		id string

		xmlError      *subsonicError
		httpCode      int
		contentType   string
		contentLength int
	}{
		{
			name: "no ID",

			xmlError: &subsonicError{Code: codeMissingParameter},
		},
		{
			name: "bad ID",

			id: "foo",

			xmlError: &subsonicError{Code: codeGeneric},
		},
		{
			name: "no files",

			id: "0",

			httpCode: http.StatusNotFound,
		},
		{
			name: "one MP3",
			db: &memoryDatabase{
				files: []string{"foo.mp3"},
			},
			fs: &memoryFilesystem{
				files: map[string]*memoryFile{
					filepath.Join(musicDirectory, "foo.mp3"): &memoryFile{
						ReadSeeker: strings.NewReader(`hello`),
					},
				},
			},

			id: "0",

			contentType:   audioMPEG,
			contentLength: 5,
		},
		{
			name: "three files",
			db: &memoryDatabase{
				files: []string{
					"foo.mp3",
					"foo/bar.ogg",
					"foo/bar/baz.flac",
				},
			},
			fs: &memoryFilesystem{
				files: map[string]*memoryFile{
					filepath.Join(musicDirectory, "foo.mp3"): &memoryFile{
						ReadSeeker: strings.NewReader(`mp3`),
					},
					filepath.Join(musicDirectory, "foo/bar.ogg"): &memoryFile{
						ReadSeeker: strings.NewReader(`ogg`),
					},
					filepath.Join(musicDirectory, "foo/bar/baz.flac"): &memoryFile{
						ReadSeeker: strings.NewReader(`flac`),
					},
				},
			},

			// ID determined by indexing algorithm
			id: "4",

			contentType:   audioFLAC,
			contentLength: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, values := configAuth()
			cfg.MusicDirectory = musicDirectory

			if tt.id != "" {
				values.Set("id", tt.id)
			}

			withServer(t, tt.db, tt.fs, cfg, func(base string) {
				res := testRequest(t, base, http.MethodGet, "/rest/stream.view", values)

				if tt.xmlError != nil {
					c := mustDecodeXML(t, res)
					if want, got := tt.xmlError.Code, c.Error.Code; want != got {
						t.Fatalf("unexpected XML error code::\n- want: %v\n-  got: %v",
							want, got)
					}

					return
				}

				if tt.httpCode != 0 {
					if want, got := tt.httpCode, res.StatusCode; want != got {
						t.Fatalf("unexpected HTTP status code:\n- want: %03d\n-  got: %03d",
							want, got)
					}

					return
				}

				if want, got := tt.contentType, res.Header.Get(contentType); want != got {
					t.Fatalf("unexpected Content-Type header:\n- want: %q\n-  got: %q",
						want, got)
				}

				if want, got := tt.contentLength, int(res.ContentLength); want != got {
					t.Fatalf("unexpected Content-Length header:\n- want: %v\n-  got: %v",
						want, got)
				}
			})
		})
	}
}

func Test_stack(t *testing.T) {
	var s stack
	s.Push("foo")
	s.Push("bar")
	s.Push("baz")

	want := []string{"baz", "bar", "foo"}

	for i, str := 0, s.Pop(); str != ""; i, str = i+1, s.Pop() {
		if want, got := want[i], str; want != got {
			t.Fatalf("unexpected string from stack:\n- want: %q\n-  got: %q",
				want, got)
		}
	}
}

// mustMusicDirectoryContainersEqual is a helper function for running subtests to
// compare to music directory containers.
func mustMusicDirectoryContainersEqual(t *testing.T, a *musicDirectoryContainer, b *musicDirectoryContainer) {
	if want, got := a.ID, b.ID; want != got {
		t.Fatalf("unexpected IDs:\n- want: %v\n-  got: %v",
			want, got)
	}

	if want, got := a.Name, b.Name; want != got {
		t.Fatalf("unexpected names:\n- want: %q\n-  got: %q",
			want, got)
	}

	if want, got := len(a.Children), len(b.Children); want != got {
		t.Fatalf("unexpected children length:\n- want: %v\n-  got: %v",
			want, got)
	}

	for i := range a.Children {
		ttChild := a.Children[i]
		child := b.Children[i]

		t.Run(ttChild.Title, func(t *testing.T) {
			ttChild.XMLName = xml.Name{}
			child.XMLName = xml.Name{}

			if want, got := ttChild, child; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected child:\n- want: %v\n-  got: %v",
					want, got)
			}
		})
	}
}

// mustIndexesEqual is a helper function for running subtests to compare two indexes.
func mustIndexesEqual(t *testing.T, a []index, b []index) {
	if want, got := len(a), len(b); want != got {
		t.Fatalf("unexpected indexes length:\n- want: %v\n-  got: %v",
			want, got)
	}

	for i := range a {
		ttIdx := a[i]
		idx := b[i]

		t.Run(ttIdx.Name, func(t *testing.T) {
			if want, got := ttIdx.Name, idx.Name; want != got {
				t.Fatalf("unexpected index name:\n- want: %v\n-  got: %v",
					want, got)
			}

			if want, got := len(ttIdx.Artists), len(idx.Artists); want != got {
				t.Fatalf("unexpected index artists:\n- want: %v\n-  got: %v",
					want, got)
			}

			for j := range ttIdx.Artists {
				ttArtist := ttIdx.Artists[j]
				artist := idx.Artists[j]

				t.Run(ttArtist.Name, func(t *testing.T) {
					if want, got := ttArtist.Name, artist.Name; want != got {
						t.Fatalf("unexpected artist name:\n- want: %v\n-  got: %v",
							want, got)
					}

					if want, got := ttArtist.ID, artist.ID; want != got {
						t.Fatalf("unexpected artist ID:\n- want: %v\n-  got: %v",
							want, got)
					}
				})
			}
		})
	}
}
