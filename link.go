package link

import (
	"io"
	"net/url"
)

// Link is the public interface for a "smart URL" which knows its destination
type Link interface {
	OriginalURL() string
	FinalURL() (*url.URL, error)
}

// Lifecycle defines common creation / destruction methods
type Lifecycle interface {
	TraverseLink(urlText string) TraversalStatus
}

// Reader defines common reader methods
type Reader interface {
	GetLink(urlText string) TraversalStatus
	HasLink(urlText string) (bool, error)
}

// Writer defines common writer methods
type Writer interface {
	WriteLink(Link) error
	DeleteLink(Link) error
}

// Store pulls together all the lifecyle, reader, and writer methods
type Store interface {
	Reader
	Writer
	io.Closer
}
