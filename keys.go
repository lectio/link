package link

import (
	"crypto/sha1"
	"fmt"
	"net/url"
)

// Keys describes the different ways link keys can be generated
type Keys interface {
	LinkKeyForURL(url *url.URL) string
	LinkKeyForURLText(urlText string) string
	LinkKey(link *Link) string
}

// MakeDefaultKeys creates a default key generator for links
func MakeDefaultKeys() Keys {
	result := new(defaultKeys)
	return result
}

type defaultKeys struct {
}

func (k defaultKeys) LinkKeyForURL(url *url.URL) string {
	if url != nil {
		return k.LinkKeyForURLText(url.String())
	}
	return "url_is_nil_in_LinkKeyForURL"
}

func (k defaultKeys) LinkKeyForURLText(urlText string) string {
	h := sha1.New()
	h.Write([]byte(urlText))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func (k defaultKeys) LinkKey(link *Link) string {
	if link != nil {
		if link.IsDestValid && link.FinalizedURL != nil {
			return k.LinkKeyForURLText(link.FinalizedURL.String())
		} else {
			return k.LinkKeyForURLText(link.OrigURLText)
		}
	}
	return "link_is_nil_in_LinkKey"
}
