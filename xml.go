package mpdsub

import (
	"encoding/xml"
	"io"
	"net/http"
)

const (
	// XMLNS is the XML namespace of a Subsonic XML response
	xmlNS = "http://subsonic.org/restapi"
	// Version is the emulated Subsonic API version
	apiVersion = "1.14.0"
)

const (
	// Possible status strings to return.
	statusOK     = "ok"
	statusFailed = "failed"

	// Possible status codes.
	codeGeneric          = 0
	codeMissingParameter = 10
	codeUnauthorized     = 40
)

// errUnauthorized indicates an incorrect username or password.
func errUnauthorized(c *container) {
	c.Status = statusFailed
	c.Error = &subsonicError{
		Code:    40,
		Message: "Wrong username or password.",
	}
}

// errMissingParameter indicates a missing required parameter.
func errMissingParameter(c *container) {
	c.Status = statusFailed
	c.Error = &subsonicError{
		Code:    10,
		Message: "Required parameter is missing.",
	}
}

// errGeneric indicates a generic error.
func errGeneric(c *container) {
	c.Status = statusFailed
	c.Error = &subsonicError{
		Code:    0,
		Message: "An error occurred.",
	}
}

const (
	// Content-Type header name and XML content type.
	contentType    = "Content-Type"
	contentTypeXML = "text/xml; charset=utf-8"
)

// writeXML writes an XML body to w after modifying it using the input function.
func writeXML(w io.Writer, fn func(c *container)) {
	c := &container{
		XMLNS:   xmlNS,
		Status:  statusOK,
		Version: apiVersion,
	}

	if fn != nil {
		fn(c)
	}

	// Set HTTP content type if available
	if rw, ok := w.(http.ResponseWriter); ok {
		rw.Header().Set(contentType, contentTypeXML)
	}

	_ = xml.NewEncoder(w).Encode(c)
}

// A container is the top-level emulated Subsonic response.
type container struct {
	// Top-level container name.
	XMLName xml.Name `xml:"subsonic-response"`

	// Attributes which are always present.
	XMLNS   string `xml:"xmlns,attr"`
	Status  string `xml:"status,attr"`
	Version string `xml:"version,attr"`

	// Error, returned on failures.
	Error *subsonicError

	Indexes        *indexesContainer
	License        *license
	MusicDirectory *musicDirectoryContainer
	MusicFolders   *musicFoldersContainer
}

// A subsonicError contains a Subsonic error, with status code and message.
type subsonicError struct {
	XMLName xml.Name `xml:"error,omitempty"`

	Code    int    `xml:"code,attr"`
	Message string `xml:"message,attr"`
}

// A license is a Subsonic license structure.
type license struct {
	XMLName xml.Name `xml:"license,omitempty"`

	Valid bool `xml:"valid,attr"`
}

// A musicFoldersContainer contains a list of emulated Subsonic music folders.
type musicFoldersContainer struct {
	XMLName xml.Name `xml:"musicFolders,omitempty"`

	MusicFolders []musicFolder `xml:"musicFolder"`
}

// A musicFolder represents an emulated Subsonic music folder.
type musicFolder struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// indexesContainer represents a Subsonic indexes container.
type indexesContainer struct {
	XMLName xml.Name `xml:"indexes,omitempty"`

	LastModified int64   `xml:"lastModified,attr"`
	Indexes      []index `xml:"index"`
}

// An index represents an alphabetical Subsonic index.
type index struct {
	XMLName xml.Name `xml:"index"`

	Name string `xml:"name,attr"`

	Artists []artist `xml:"artist"`
}

// An artist represents an emulated Subsonic artist.
type artist struct {
	XMLName xml.Name `xml:"artist,omitempty"`

	Name string `xml:"name,attr"`
	ID   string `xml:"id,attr"`
}

// A musicDirectoryContainer contains a list of emulated Subsonic music folders.
type musicDirectoryContainer struct {
	XMLName xml.Name `xml:"directory,omitempty"`

	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`

	Children []child `xml:"child"`
}

// A child is any item displayed to Subsonic when browsing using getMusicDirectory.
type child struct {
	XMLName xml.Name `xml:"child,omitempty"`

	ID       string `xml:"id,attr"`
	Title    string `xml:"title,attr"`
	Album    string `xml:"album,attr"`
	Artist   string `xml:"artist,attr"`
	IsDir    bool   `xml:"isDir,attr"`
	CoverArt int    `xml:"coverArt,attr"`
	Created  string `xml:"created,attr"`
}
