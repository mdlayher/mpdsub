package mpdsub

import (
	"reflect"
	"testing"

	"github.com/fhs/gompd/mpd"
)

func Test_indexFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		out   []indexedFile
	}{
		{
			name: "no files",
		},
		{
			name: "one file",
			files: []string{
				"foo.mp3",
			},
			out: []indexedFile{{
				ID:   0,
				Name: "foo.mp3",
				Dir:  false,
			}},
		},
		{
			name: "three files",
			files: []string{
				"bar.mp3",
				"baz.mp3",
				"foo.mp3",
			},
			out: []indexedFile{
				{
					ID:   0,
					Name: "bar.mp3",
				},
				{
					ID:   1,
					Name: "baz.mp3",
				},
				{
					ID:   2,
					Name: "foo.mp3",
				},
			},
		},
		{
			name: "two files, different folders",
			files: []string{
				"bar/bar.mp3",
				"foo.mp3",
			},
			out: []indexedFile{
				{
					ID:   0,
					Name: "bar",
					Dir:  true,
				},
				{
					ID:   1,
					Name: "bar/bar.mp3",
				},
				{
					ID:   2,
					Name: "foo.mp3",
				},
			},
		},
		{
			name: "two files, nested folders",
			files: []string{
				"bar/baz/qux/bar.mp3",
				"foo.mp3",
			},
			out: []indexedFile{
				{
					ID:   0,
					Name: "bar",
					Dir:  true,
				},
				{
					ID:   1,
					Name: "bar/baz",
					Dir:  true,
				},
				{
					ID:   2,
					Name: "bar/baz/qux",
					Dir:  true,
				},
				{
					ID:   3,
					Name: "bar/baz/qux/bar.mp3",
				},
				{
					ID:   4,
					Name: "foo.mp3",
				},
			},
		},
		{
			name: "multiple artists and albums",
			files: []string{
				"Boston/1976 - Boston/01 - More Than A Feeling.flac",
				"Boston/1976 - Boston/02 - Peace Of Mind.flac",
				"Jimmy Eat World/1999 - Clarity/01 - Table for Glasses.flac",
				"Jimmy Eat World/1999 - Clarity/02 - Lucky Denver Mint.flac",
				"Jimmy Eat World/2001 - Bleed American/01 - Bleed American.flac",
			},
			out: []indexedFile{
				{
					ID:   0,
					Name: "Boston",
					Dir:  true,
				},
				{
					ID:   1,
					Name: "Boston/1976 - Boston",
					Dir:  true,
				},
				{
					ID:   2,
					Name: "Boston/1976 - Boston/01 - More Than A Feeling.flac",
				},
				{
					ID:   3,
					Name: "Boston/1976 - Boston/02 - Peace Of Mind.flac",
				},
				{
					ID:   4,
					Name: "Jimmy Eat World",
					Dir:  true,
				},
				{
					ID:   5,
					Name: "Jimmy Eat World/1999 - Clarity",
					Dir:  true,
				},
				{
					ID:   6,
					Name: "Jimmy Eat World/1999 - Clarity/01 - Table for Glasses.flac",
				},
				{
					ID:   7,
					Name: "Jimmy Eat World/1999 - Clarity/02 - Lucky Denver Mint.flac",
				},
				{
					ID:   8,
					Name: "Jimmy Eat World/2001 - Bleed American",
					Dir:  true,
				},
				{
					ID:   9,
					Name: "Jimmy Eat World/2001 - Bleed American/01 - Bleed American.flac",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := indexFiles(tt.files)

			if want, got := tt.out, out; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected output:\n- want: %v\n-  got: %v", want, got)
			}
		})
	}
}

func Test_filterFiles(t *testing.T) {
	tests := []struct {
		name  string
		in    []indexedFile
		start int
		out   []indexedFile
	}{
		{
			name: "no input files",
			out:  []indexedFile{},
		},
		{
			name: "start index out of bounds",
			in: []indexedFile{{
				Name: "foo",
			}},
			start: 1,
			out:   []indexedFile{},
		},
		{
			name: "filter all items; first refers to self, second with different prefix",
			in: []indexedFile{
				{Name: "foo"},
				{Name: "bar"},
			},
			start: 0,
			out:   []indexedFile{},
		},
		{
			name: "one directory",
			in: []indexedFile{
				{Name: "foo"},
				{Name: "foo/bar"},
				{Name: "bar"},
			},
			start: 0,
			out: []indexedFile{
				{Name: "foo/bar"},
			},
		},
		{
			name: "one directory and file, children ignored",
			in: []indexedFile{
				{Name: "foo"},
				{Name: "foo/bar"},
				{Name: "foo/foo.mp3"},
				{Name: "foo/bar/bar.mp3"},
				{Name: "bar"},
			},
			start: 0,
			out: []indexedFile{
				{Name: "foo/bar"},
				{Name: "foo/foo.mp3"},
			},
		},
		{
			name: "second set of directories with early stop",
			in: []indexedFile{
				{Name: "foo"},
				{Name: "foo/bar"},
				{Name: "foo/foo.mp3"},
				{Name: "foo/bar/bar.mp3"},
				{Name: "bar"},
				{Name: "bar/baz"},
				{Name: "bar/baz/qux.mp3"},
				{Name: "bar/baz.mp3"},
				{Name: "baz"},
			},
			start: 4,
			out: []indexedFile{
				{Name: "bar/baz"},
				{Name: "bar/baz.mp3"},
			},
		},
		{
			name: "second set of directories reach end of list",
			in: []indexedFile{
				{Name: "foo"},
				{Name: "foo/bar"},
				{Name: "foo/foo.mp3"},
				{Name: "foo/bar/bar.mp3"},
				{Name: "bar"},
				{Name: "bar/baz"},
				{Name: "bar/baz/qux.mp3"},
				{Name: "bar/baz.mp3"},
			},
			start: 4,
			out: []indexedFile{
				{Name: "bar/baz"},
				{Name: "bar/baz.mp3"},
			},
		},
		{
			name: "all files filtered, because no files with new prefix",
			in: []indexedFile{{
				Name: "foo",
			}},
			out: []indexedFile{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := filterFiles(tt.in, tt.start)

			if want, got := tt.out, out; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected output:\n- want: %v\n-  got: %v", want, got)
			}
		})
	}
}

func Test_tagFiles(t *testing.T) {
	tests := []struct {
		name string
		db   database
		in   []indexedFile
		out  []metadataFile
	}{
		{
			name: "no files",
			in:   []indexedFile{},
			out:  []metadataFile{},
		},
		{
			name: "one file",
			db: &memoryDatabase{
				attrs: map[string]mpd.Attrs{
					"foo.mp3": mpd.Attrs{
						"ARTIST": "Baz",
						"ALBUM":  "Bar",
						"TITLE":  "Foo",
					},
				},
			},
			in: []indexedFile{{
				ID:   0,
				Name: "foo.mp3",
				Dir:  false,
			}},
			out: []metadataFile{{
				indexedFile: indexedFile{
					ID:   0,
					Name: "foo.mp3",
					Dir:  false,
				},

				Artist: "Baz",
				Album:  "Bar",
				Title:  "Foo",
			}},
		},
		{
			name: "nested directories, directory with music inherits tags",
			db: &memoryDatabase{
				attrs: map[string]mpd.Attrs{
					"foo/bar/bar.mp3": mpd.Attrs{
						"ARTIST": "Baz",
						"ALBUM":  "Bar",
						"TITLE":  "Foo",
					},
				},
			},
			in: []indexedFile{
				{
					ID:   0,
					Name: "foo",
					Dir:  true,
				},
				{
					ID:   1,
					Name: "foo/bar",
					Dir:  true,
				},
				{
					ID:   2,
					Name: "foo/bar/bar.mp3",
					Dir:  false,
				},
			},
			out: []metadataFile{
				{
					indexedFile: indexedFile{
						ID:   0,
						Name: "foo",
						Dir:  true,
					},

					Title: "foo",
				},
				{
					indexedFile: indexedFile{
						ID:   1,
						Name: "foo/bar",
						Dir:  true,
					},

					Artist: "Baz",
					Album:  "Bar",
					Title:  "Bar",
				},
				{
					indexedFile: indexedFile{
						ID:   2,
						Name: "foo/bar/bar.mp3",
						Dir:  false,
					},

					Artist: "Baz",
					Album:  "Bar",
					Title:  "Foo",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tagFiles(tt.db, tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tt.out, out; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected output:\n- want: %v\n-  got: %v", want, got)
			}
		})
	}
}
