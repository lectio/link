package cache

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/lectio/link"
)

type tempCache struct {
	fileCache        *fileCache
	removeAllOnClose bool
}

// MakeTempCache creates an instance of a cache, which stores links on disk, in a temp directory
func MakeTempCache(keys link.Keys, clpr link.CleanLinkParamsRule, ilr link.IgnoreLinkRule, dr link.DestinationRule, removeAllOnClose bool) (Cache, error) {
	tempPath, err := ioutil.TempDir("", "links")
	if err != nil {
		return nil, err
	}
	result := new(tempCache)
	fc, fcErr := MakeFileCache(tempPath, false, keys, clpr, ilr, dr)
	if fcErr != nil {
		return nil, fcErr
	}
	result.fileCache = fc.(*fileCache)
	result.removeAllOnClose = removeAllOnClose
	return result, nil
}

func (c tempCache) Harvest(urlText string) (*link.Link, error) {
	return c.fileCache.Harvest(urlText)
}

func (c tempCache) Get(urlText string) (*link.Link, error) {
	return c.fileCache.Get(urlText)
}

func (c tempCache) Find(urlText string) (link *link.Link, found bool, expired bool, err error) {
	return c.Find(urlText)
}

func (c tempCache) Save(link *link.Link, autoExpire time.Duration) error {
	return c.Save(link, autoExpire)
}

func (c tempCache) Close() error {
	if c.removeAllOnClose {
		return os.RemoveAll(c.fileCache.path)
	}
	return nil
}
