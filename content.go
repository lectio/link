package link

import (
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Content manages the kind of content of a link
type Content struct {
	URL                          *url.URL          `json:"url"`
	ContentType                  string            `json:"contentType"`
	MediaType                    string            `json:"mediaType"`
	MediaTypeParams              map[string]string `json:"mediaTypeParams"`
	MediaTypeError               error             `json:"mediaTypeError,omitempty"`
	HTMLParsed                   bool              `json:"htmlParsed"`
	HTMLParseError               error             `json:"htmlParseError,omitempty"`
	IsHTMLRedirect               bool              `json:"isHTMLRedirect"`
	MetaRefreshTagContentURLText string            `json:"metaRefreshTagContentURLText"` // if IsHTMLRedirect is true, then this is the value after url= in something like <meta http-equiv='refresh' content='delay;url='>
	MetaPropertyTags             map[string]string `json:"metaPropertyTags"`             // if IsHTML() is true, a collection of all meta data like <meta property="og:site_name" content="Netspective" /> or <meta name="twitter:title" content="text" />
	Attachment                   *Attachment       `json:"attachment"`
}

// MakeContent will figure out what kind of destination content we're dealing with
func MakeContent(url *url.URL, resp *http.Response, destRule DestinationRule) *Content {
	result := new(Content)
	result.MetaPropertyTags = make(map[string]string)
	result.URL = url
	result.ContentType = resp.Header.Get("Content-Type")
	if len(result.ContentType) > 0 {
		result.MediaType, result.MediaTypeParams, result.MediaTypeError = mime.ParseMediaType(result.ContentType)
		if result.MediaTypeError != nil {
			return result
		}
		if result.IsHTML() && (destRule.FollowRedirectsInDestinationHTMLContent(url) || destRule.ParseMetaDataInDestinationHTMLContent(url)) {
			result.parsePageMetaData(url, resp)
			result.HTMLParsed = true
			return result
		}
	}

	// If we get to here it means that we need to download the content to inspect it any further.
	// We download it first because it's possible we want to retain it for later use.
	downloadAttachment, destFileName := destRule.DownloadAttachmentsFromDestination(url)
	if downloadAttachment {
		if len(destFileName) == 0 {
			result.Attachment = downloadTemp(url, resp, "link-Attachment-")
		} else {
			result.Attachment = download(url, resp, destFileName)
		}
	}
	return result
}

// metaRefreshContentRegEx is used to match the 'content' attribute in a tag like this:
//   <meta http-equiv="refresh" content="2;url=https://www.google.com">
var metaRefreshContentRegEx = regexp.MustCompile(`^(\d?)\s?;\s?url=(.*)$`)

func (c *Content) parsePageMetaData(url *url.URL, resp *http.Response) error {
	doc, parseError := html.Parse(resp.Body)
	if parseError != nil {
		c.HTMLParseError = parseError
		return parseError
	}
	defer resp.Body.Close()

	var inHead bool
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "head") {
			inHead = true
		}
		if inHead && n.Type == html.ElementNode && strings.EqualFold(n.Data, "meta") {
			for _, attr := range n.Attr {
				if strings.EqualFold(attr.Key, "http-equiv") && strings.EqualFold(strings.TrimSpace(attr.Val), "refresh") {
					for _, attr := range n.Attr {
						if strings.EqualFold(attr.Key, "content") {
							contentValue := strings.TrimSpace(attr.Val)
							parts := metaRefreshContentRegEx.FindStringSubmatch(contentValue)
							if parts != nil && len(parts) == 3 {
								// the first part is the entire match
								// the second and third parts are the delay and URL
								// See for explanation: http://redirectdetective.com/redirection-types.html
								c.IsHTMLRedirect = true
								c.MetaRefreshTagContentURLText = parts[2]
							}
						}
					}
				}
				if strings.EqualFold(attr.Key, "property") || strings.EqualFold(attr.Key, "name") {
					propertyName := attr.Val
					for _, attr := range n.Attr {
						if strings.EqualFold(attr.Key, "content") {
							c.MetaPropertyTags[propertyName] = attr.Val
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return nil
}

// IsValid returns true if there are no errors
func (c Content) IsValid() bool {
	if c.MediaTypeError != nil {
		return false
	}

	if c.Attachment != nil {
		return c.Attachment.IsValid()
	}

	return true
}

// IsHTML returns true if this is HTML content
func (c Content) IsHTML() bool {
	return c.MediaType == "text/html"
}

// GetOpenGraphMetaTag returns the value and true if og:key was found
func (c Content) GetOpenGraphMetaTag(key string) (string, bool) {
	result, ok := c.MetaPropertyTags["og:"+key]
	return result, ok
}

// GetTwitterMetaTag returns the value and true if og:key was found
func (c Content) GetTwitterMetaTag(key string) (string, bool) {
	result, ok := c.MetaPropertyTags["twitter:"+key]
	return result, ok
}

// IsContentBasedRedirect returns true if redirect was requested through via <meta http-equiv='refresh' content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (c Content) IsContentBasedRedirect() (bool, string) {
	return c.IsHTMLRedirect, c.MetaRefreshTagContentURLText
}

// WasDownloaded returns true if content was downloaded
func (c Content) WasDownloaded() bool {
	return c.Attachment != nil
}
