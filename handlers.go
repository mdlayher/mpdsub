package mpdsub

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// getLicense returns a license that is always valid.
func (s *Server) getLicense(w http.ResponseWriter, r *http.Request) {
	writeXML(w, func(c *container) {
		// A license that indicates valid "true" allows Subsonic
		// clients to connect to this server
		c.License = &license{Valid: true}
	})
}

// getIndexes returns a set of top-level indexes that indicate the top-level
// items and directories.
func (s *Server) getIndexes(w http.ResponseWriter, r *http.Request) {
	fs, err := s.db.List("file")
	if err != nil {
		s.logf("error listing files from mpd for building indexes: %v", err)
		writeXML(w, errGeneric)
		return
	}
	files := indexFiles(fs)

	writeXML(w, func(c *container) {
		c.Indexes = &indexesContainer{
			LastModified: time.Now().Unix(),
		}

		// Incremented whenever it's time to create a new index for a new
		// initial letter
		idx := -1

		var indexes []index

		// A set of initial characters, used to deduplicate the addition of
		// nwe indexes
		seenChars := make(map[rune]struct{}, 0)

		for _, f := range files {
			// Filter any non-top level items
			if strings.Contains(f.Name, string(os.PathSeparator)) {
				continue
			}

			// Initial rune is used to create an index name
			c, _ := utf8.DecodeRuneInString(f.Name)
			name := string(c)

			// If initial rune is a digit, put index under a numeric section
			if unicode.IsDigit(c) {
				c = '#'
				name = "#"
			}

			// If a new rune appears, create a new index for it
			if _, ok := seenChars[c]; !ok {
				seenChars[c] = struct{}{}
				indexes = append(indexes, index{Name: name})
				idx++
			}

			indexes[idx].Artists = append(indexes[idx].Artists, artist{
				Name: f.Name,
				ID:   strconv.Itoa(f.ID),
			})
		}

		c.Indexes.Indexes = indexes
	})
}

// getMusicDirectory returns the contents of a single music directory.
func (s *Server) getMusicDirectory(w http.ResponseWriter, r *http.Request) {
	qID := r.URL.Query().Get("id")
	if qID == "" {
		writeXML(w, errMissingParameter)
		return
	}

	id, err := strconv.Atoi(qID)
	if err != nil {
		writeXML(w, errGeneric)
		return
	}

	fs, err := s.db.List("file")
	if err != nil {
		s.logf("error listing files from mpd for getting music directory: %v", err)
		writeXML(w, errGeneric)
		return
	}

	files, err := tagFiles(s.db, filterFiles(indexFiles(fs), id))
	if err != nil {
		log.Println(err)
		s.logf("error tagging files from mpd for getting music directory: %v", err)
		writeXML(w, errGeneric)
		return
	}

	// No files matching criteria
	if len(files) == 0 {
		http.NotFound(w, r)
		return
	}

	var children []child
	for _, f := range files {
		ext := strings.TrimPrefix(filepath.Ext(f.Name), ".")
		children = append(children, child{
			ID:     strconv.Itoa(f.ID),
			Album:  f.Album,
			Artist: f.Artist,
			IsDir:  f.Dir,
			Suffix: ext,
			Title:  f.Title,
		})
	}

	writeXML(w, func(c *container) {
		c.MusicDirectory = &musicDirectoryContainer{
			ID:       strconv.Itoa(id),
			Name:     files[0].Name,
			Children: children,
		}
	})
}

// getMusicFolders returns the location of MPD's music directory.
func (s *Server) getMusicFolders(w http.ResponseWriter, r *http.Request) {
	writeXML(w, func(c *container) {
		c.MusicFolders = &musicFoldersContainer{
			MusicFolders: []musicFolder{{
				ID:   0,
				Name: filepath.Base(s.cfg.MusicDirectory),
			}},
		}
	})
}

// ping returns an empty response to indicate the server is working.
func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	writeXML(w, nil)
}

// stream opens a file for streaming, and serves it to a client.
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	qID := r.URL.Query().Get("id")
	if qID == "" {
		writeXML(w, errMissingParameter)
		return
	}

	id, err := strconv.Atoi(qID)
	if err != nil {
		writeXML(w, errGeneric)
		return
	}

	fs, err := s.db.List("file")
	if err != nil {
		s.logf("error listing files from mpd for streaming: %v", err)
		writeXML(w, errGeneric)
		return
	}
	files := indexFiles(fs)

	// Don't allow out of bounds slice access
	if id >= len(files) {
		http.NotFound(w, r)
		return
	}

	p := filepath.Join(s.cfg.MusicDirectory, files[id].Name)

	f, err := s.fs.Open(p)
	if err != nil {
		s.logf("error opening file for streaming: %q", p)
		writeXML(w, errGeneric)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		s.logf("error stat'ing file for streaming: %q", p)
		writeXML(w, errGeneric)
		return
	}

	http.ServeContent(w, r, p, stat.ModTime(), f)
}

// A stack is a stack data structure for strings.
type stack []string

// Push pushes an element on top of the stack.
func (s *stack) Push(str string) {
	*s = append(*s, str)
}

// Pop pops the top element off of the stack.  If no elements remain,
// empty string is returned.
func (s *stack) Pop() string {
	if len(*s) == 0 {
		return ""
	}

	str := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return str
}
