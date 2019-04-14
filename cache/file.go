package cache

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/lectio/link"
)

const defaultFilePerm os.FileMode = 0644

type fileCache struct {
	path                string
	extension           string
	perm                os.FileMode
	keys                link.Keys
	cleanLinkParamsRule link.CleanLinkParamsRule
	ignoreLinkRule      link.IgnoreLinkRule
	destinationRule     link.DestinationRule
}

// MakeFileCache creates an instance of a cache, which stores links on disk, in a named path
func MakeFileCache(path string, createPath bool, keys link.Keys, clpr link.CleanLinkParamsRule, ilr link.IgnoreLinkRule, dr link.DestinationRule) (Cache, error) {
	if createPath {
		if err := os.MkdirAll(path, defaultFilePerm); err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("lectio link cache directory does not exist: %q", path)
	}

	cache := new(fileCache)
	cache.perm = defaultFilePerm
	cache.path = path
	cache.extension = ".json"
	cache.keys = keys
	cache.cleanLinkParamsRule = clpr
	cache.ignoreLinkRule = ilr
	cache.destinationRule = dr
	return cache, nil
}

func (c fileCache) Harvest(urlText string) (*link.Link, error) {
	return link.HarvestLink(urlText, c.cleanLinkParamsRule, c.ignoreLinkRule, c.destinationRule), nil
}

func (c fileCache) Get(urlText string) (*link.Link, error) {
	link, found, expired, err := c.Find(urlText)
	if err != nil {
		return nil, err
	}

	if found && !expired {
		return link, err
	}

	link, err = c.Harvest(urlText)
	if err != nil {
		return nil, err
	}
	c.Save(link, 0)
	return link, nil
}

func (c fileCache) Find(urlText string) (*link.Link, bool, bool, error) {
	key := c.keys.PrimaryKeyForURLText(urlText)
	fileName := path.Join(c.path, key+c.extension)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil, false, true, nil
	}

	file, openErr := os.Open(fileName)
	if openErr != nil {
		return nil, false, true, openErr
	}

	bytes, readErr := ioutil.ReadAll(file)
	if readErr != nil {
		return nil, false, true, readErr
	}

	var link link.Link
	parseErr := json.Unmarshal(bytes, &link)
	if parseErr != nil {
		return nil, false, true, parseErr
	}

	return &link, true, false, nil
}

func (c fileCache) Save(link *link.Link, autoExpire time.Duration) error {
	linkJSON, marshErr := json.Marshal(link)
	if marshErr != nil {
		return marshErr
	}
	return ioutil.WriteFile(path.Join(c.path, link.PrimaryKey(c.keys)+c.extension), linkJSON, c.perm)
}

func (c fileCache) Close() error {
	return nil
}
