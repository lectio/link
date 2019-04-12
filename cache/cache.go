package cache

import (
	"time"

	"github.com/lectio/link"
)

// Cache allows storing and retrieving links from disk, RAM, etc.
type Cache interface {
	Harvest(urlText string) (*link.Link, error)
	Get(urlText string) (*link.Link, error)
	Find(urlText string) (link *link.Link, found bool, expired bool, err error)
	Save(link *link.Link, autoExpire time.Duration) error
	Close() error
}
