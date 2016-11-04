package mpdsub

import (
	"os"
	"path/filepath"
	"strings"
)

// An indexedFile is a file with an associated ID, name, and a boolean to
// indicate if it is a directory or not.
type indexedFile struct {
	ID   int
	Name string
	Dir  bool
}

// A metadataFile is an indexedFile with metadata attached.
type metadataFile struct {
	indexedFile

	Artist string
	Album  string
	Title  string
}

// indexFiles builds a slice of indexedFiles from a file list returned by
// MPD.  Each file and unique directory is given an ID and a boolean value
// to indicate if it's a file or directory.
func indexFiles(files []string) []indexedFile {
	// Track duplicate directories
	seen := make(map[string]struct{}, 0)

	// Index to assign to files
	var idx int

	var out []indexedFile
	for _, f := range files {
		// Track directories encountered using a stack
		var dirs stack

		// While a new directory is available for this filename, iterate and
		// check it for uniqueness
		for d := filepath.Dir(f); d != "."; d = filepath.Dir(d) {
			// Has directory already been seen?
			if _, ok := seen[d]; ok {
				continue
			}

			// New directory, add to set and push onto stack
			seen[d] = struct{}{}
			dirs.Push(d)
		}

		// While directories are available on the stack, pop them off and
		// give them an index
		for d := dirs.Pop(); d != ""; d = dirs.Pop() {
			out = append(out, indexedFile{
				ID:   idx,
				Name: d,
				Dir:  true,
			})
			idx++
		}

		// Give each normal file an index
		out = append(out, indexedFile{
			ID:   idx,
			Name: f,
			Dir:  false,
		})
		idx++
	}

	return out
}

// filterFiles filters an input slice of indexedFiles and produces a
// filtered output slice containing all of the items which belong in a given
// directory, specified using its index in start.
func filterFiles(files []indexedFile, start int) []indexedFile {
	if start >= len(files) {
		return []indexedFile{}
	}

	var copied bool
	var out []indexedFile
	for i := range files[start:] {
		// If we encounter a file that does not have the same prefix as the
		// initial starting file, copy what we need and break the loop
		if !strings.HasPrefix(files[start+i].Name, files[start].Name) {
			out = make([]indexedFile, len(files[start:start+i]))
			copy(out, files[start:start+i])
			copied = true

			break
		}
	}

	// If files were not copied in loop, must have run off the end of the list
	if !copied {
		out = make([]indexedFile, len(files[start:]))
		copy(out, files[start:])
	}

	// Track number of separators in initial item to determine how many we
	// can allow to retrieve items in the current directory, but not items
	// from child directories
	sepCount := strings.Count(out[0].Name, string(os.PathSeparator))

	var filter []indexedFile
	for _, f := range out {
		// Filter items from child directories
		if strings.Count(f.Name, string(os.PathSeparator)) > sepCount+1 {
			continue
		}

		filter = append(filter, f)
	}

	// Trim first item because it refers to itself
	return filter[1:]
}

// tagFiles attaches metadata to an input slice of indexedFiles and returns
// a slice of metadataFiles.  Tag information is looked up using the input
// database.
func tagFiles(db database, files []indexedFile) ([]metadataFile, error) {
	// Cache directories so metadata can be applied to them in a second loop
	cache := make(map[string]metadataFile, 0)
	out := make([]metadataFile, 0, len(files))
	for _, f := range files {
		// Give directories a default name of the last element of their path
		if f.Dir {
			out = append(out, metadataFile{
				indexedFile: f,
				Title:       filepath.Base(f.Name),
			})
			continue
		}

		attrs, err := db.ReadComments(f.Name)
		if err != nil {
			return nil, err
		}

		// Create fileMetadata using indexedFile, adding tags read from
		// database to metadata
		newf := metadataFile{
			indexedFile: f,

			Artist: attrs["ARTIST"],
			Album:  attrs["ALBUM"],
			Title:  attrs["TITLE"],
		}

		out = append(out, newf)

		// Add this metadata to the cache so the directory can be tagged later
		dir := filepath.Dir(f.Name)
		cache[dir] = newf
	}

	for i, f := range out {
		// Skip top-level and non-directories
		if !strings.Contains(f.Name, string(os.PathSeparator)) {
			continue
		}
		if !f.Dir {
			continue
		}

		// Tag directories with metadata if available
		if ff, ok := cache[f.Name]; ok {
			out[i].Artist = ff.Artist
			out[i].Album = ff.Album
			out[i].Title = ff.Album
		}
	}

	return out, nil
}
