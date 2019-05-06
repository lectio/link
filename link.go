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

// ManagedLink is the public interface for a "managed smart URL" which knows its destination and policy
type ManagedLink interface {
	Link
	Issues() Issues
	Ignore() (bool, string)
}

// Lifecycle defines common creation / destruction methods
type Lifecycle interface {
	HarvestLink(urlText string) (Link, Issue)
}

// Reader defines common reader methods
type Reader interface {
	GetLink(urlText string) (Link, Issue)
	HasLink(urlText string) (bool, Issue)
}

// Writer defines common writer methods
type Writer interface {
	WriteLink(Link) Issue
	DeleteLink(Link) Issue
}

// Store pulls together all the lifecyle, reader, and writer methods
type Store interface {
	Reader
	Writer
	io.Closer
}
