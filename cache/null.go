package cache

import (
	"time"

	"github.com/lectio/link"
)

type nullCache struct {
	cleanLinkParamsRule link.CleanLinkParamsRule
	ignoreLinkRule      link.IgnoreLinkRule
	destinationRule     link.DestinationRule
}

// MakeNullCache creates an instance of a "noop" cache, which always runs Harvest (no storage)
func MakeNullCache(clpr link.CleanLinkParamsRule, ilr link.IgnoreLinkRule, dr link.DestinationRule) Cache {
	cache := new(nullCache)
	cache.cleanLinkParamsRule = clpr
	cache.ignoreLinkRule = ilr
	cache.destinationRule = dr
	return cache
}

func (c nullCache) Harvest(urlText string) (*link.Link, error) {
	return link.HarvestLink(urlText, c.cleanLinkParamsRule, c.ignoreLinkRule, c.destinationRule), nil
}

func (c nullCache) Get(urlText string) (*link.Link, error) {
	return c.Harvest(urlText)
}

func (c nullCache) Find(urlText string) (link *link.Link, found bool, expired bool, err error) {
	return nil, false, true, nil
}

func (c nullCache) Save(link *link.Link, autoExpire time.Duration) error {
	return nil
}

func (c nullCache) Close() error {
	return nil
}
