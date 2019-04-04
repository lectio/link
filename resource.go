package link

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	filetype "github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
	"golang.org/x/net/html"
)

// DownloadedContent manages any content that was downloaded for further inspection
type DownloadedContent struct {
	url           *url.URL
	destPath      string
	downloadError error
	fileTypeError error
	fileType      types.Type
}

// Delete removes the file that was downloaded
func (dc *DownloadedContent) Delete() {
	os.Remove(dc.destPath)
}

// downloadContent will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadContent(url *url.URL, resp *http.Response) *DownloadedContent {
	destFile, err := ioutil.TempFile(os.TempDir(), "harvester-dl-")

	result := new(DownloadedContent)
	result.url = url
	if err != nil {
		result.downloadError = err
		return result
	}

	defer destFile.Close()
	defer resp.Body.Close()
	result.destPath = destFile.Name()
	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		result.downloadError = err
		return result
	}
	destFile.Close()

	// Open the just-downloaded file again since it was closed already
	file, err := os.Open(result.destPath)
	if err != nil {
		result.fileTypeError = err
		return result
	}

	// We only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	file.Read(head)
	file.Close()

	result.fileType, result.fileTypeError = filetype.Match(head)
	if result.fileTypeError == nil {
		// change the extension so that it matches the file type we found
		currentPath := result.destPath
		currentExtension := path.Ext(currentPath)
		newPath := currentPath[0:len(currentPath)-len(currentExtension)] + "." + result.fileType.Extension
		os.Rename(currentPath, newPath)
		result.destPath = newPath
	}

	return result
}

// InspectedContent manages the kind of content was inspected
type InspectedContent struct {
	url                          *url.URL
	contentType                  string
	mediaType                    string
	mediaTypeParams              map[string]string
	mediaTypeError               error
	htmlParseError               error
	isHTMLRedirect               bool
	metaRefreshTagContentURLText string            // if IsHTMLRedirect is true, then this is the value after url= in something like <meta http-equiv='refresh' content='delay;url='>
	metaPropertyTags             map[string]string // if IsHTML() is true, a collection of all meta data like <meta property="og:site_name" content="Netspective" /> or <meta name="twitter:title" content="text" />
	downloaded                   *DownloadedContent
}

// InspectCurationTarget will figure out what kind of destination content we're dealing with
func inspectContent(url *url.URL, resp *http.Response) *InspectedContent {
	result := new(InspectedContent)
	result.metaPropertyTags = make(map[string]string)
	result.url = url
	result.contentType = resp.Header.Get("Content-Type")
	if len(result.contentType) > 0 {
		result.mediaType, result.mediaTypeParams, result.mediaTypeError = mime.ParseMediaType(result.contentType)
		if result.mediaTypeError != nil {
			return result
		}
		if result.IsHTML() {
			result.parsePageMetaData(url, resp)
			return result
		}
	}

	// If we get to here it means that we need to download the content to inspect it.
	// We download it first because it's possible we want to retain it for later use.
	result.downloaded = downloadContent(url, resp)
	return result
}

// metaRefreshContentRegEx is used to match the 'content' attribute in a tag like this:
//   <meta http-equiv="refresh" content="2;url=https://www.google.com">
var metaRefreshContentRegEx = regexp.MustCompile(`^(\d?)\s?;\s?url=(.*)$`)

func (c *InspectedContent) parsePageMetaData(url *url.URL, resp *http.Response) error {
	doc, parseError := html.Parse(resp.Body)
	if parseError != nil {
		c.htmlParseError = parseError
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
								c.isHTMLRedirect = true
								c.metaRefreshTagContentURLText = parts[2]
							}
						}
					}
				}
				if strings.EqualFold(attr.Key, "property") || strings.EqualFold(attr.Key, "name") {
					propertyName := attr.Val
					for _, attr := range n.Attr {
						if strings.EqualFold(attr.Key, "content") {
							c.metaPropertyTags[propertyName] = attr.Val
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

// IsValid returns true if this there are no errors
func (c InspectedContent) IsValid() bool {
	if c.mediaTypeError != nil {
		return false
	}

	if c.downloaded != nil {
		if c.downloaded.downloadError != nil {
			return false
		}
		if c.downloaded.fileTypeError != nil {
			return false
		}
	}

	return true
}

// IsHTML returns true if this is HTML content
func (c InspectedContent) IsHTML() bool {
	return c.mediaType == "text/html"
}

// GetOpenGraphMetaTag returns the value and true if og:key was found
func (c InspectedContent) GetOpenGraphMetaTag(key string) (string, bool) {
	result, ok := c.metaPropertyTags["og:"+key]
	return result, ok
}

// GetTwitterMetaTag returns the value and true if og:key was found
func (c InspectedContent) GetTwitterMetaTag(key string) (string, bool) {
	result, ok := c.metaPropertyTags["twitter:"+key]
	return result, ok
}

// WasDownloaded returns true if content was downloaded for inspection
func (c InspectedContent) WasDownloaded() bool {
	return c.downloaded != nil
}

// IsHTMLRedirect returns true if redirect was requested through via <meta http-equiv='refresh' content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (c InspectedContent) IsHTMLRedirect() (bool, string) {
	return c.isHTMLRedirect, c.metaRefreshTagContentURLText
}

// Resource tracks a single URL that was curated or discovered in content.
// Discovered URLs are validated, follow their redirects, and may have
// query parameters "cleaned" (if instructed).
type Resource struct {
	// TODO consider adding source information (e.g. tweet, e-mail, etc.) and embed style (e.g. text, HTML <a> tag, etc.)
	harvestedOn       time.Time
	origURLtext       string
	origResource      *Resource
	isURLValid        bool
	isDestValid       bool
	httpStatusCode    int
	isURLIgnored      bool
	ignoreReason      string
	isURLCleaned      bool
	isURLAttachment   bool
	resolvedURL       *url.URL
	cleanedURL        *url.URL
	finalURL          *url.URL
	globallyUniqueKey string
	inspectionResults *InspectedContent
}

// OriginalURLText returns the URL as it was discovered, with no alterations
func (r *Resource) OriginalURLText() string {
	return r.origURLtext
}

// ReferredByResource returns the original resource that referred this one,
// which is only non-nil when this resource was an HTML (not HTTP) redirect
func (r *Resource) ReferredByResource() *Resource {
	return r.origResource
}

// IsValid indicates whether (a) the original URL was parseable and (b) whether
// the destination is valid -- meaning not a 404 or something else
func (r *Resource) IsValid() (bool, bool) {
	return r.isURLValid, r.isDestValid
}

// IsIgnored indicates whether the URL should be ignored based on harvesting rules.
// Discovered URLs may be ignored for a variety of reasons using a list of Regexps.
func (r *Resource) IsIgnored() (bool, string) {
	return r.isURLIgnored, r.ignoreReason
}

// IsCleaned indicates whether URL query parameters were removed and the new "cleaned" URL
func (r *Resource) IsCleaned() (bool, *url.URL) {
	return r.isURLCleaned, r.cleanedURL
}

// GetURLs returns the final (most useful), originally resolved, and "cleaned" URLs
func (r *Resource) GetURLs() (*url.URL, *url.URL, *url.URL) {
	return r.finalURL, r.resolvedURL, r.cleanedURL
}

// IsHTMLRedirect returns true if redirect was requested through via <meta http-equiv='refresh' content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (r *Resource) IsHTMLRedirect() (bool, string) {
	ir := r.inspectionResults
	if ir != nil {
		return ir.IsHTMLRedirect()
	}
	return false, ""
}

// InspectionResults returns the inspected or downloaded content
func (r Resource) InspectionResults() *InspectedContent {
	return r.inspectionResults
}

// GloballyUniqueKey returns a hash for the URL that can uniquely identify this resource
func (r Resource) GloballyUniqueKey() string {
	return r.globallyUniqueKey
}

// cleanResource checks to see if there are any parameters that should be removed (e.g. UTM_*)
func cleanResource(url *url.URL, rule CleanResourceParamsRule) (bool, *url.URL) {
	if !rule.CleanResourceParams(url) {
		return false, nil
	}

	// make a copy because we're planning on changing the URL params
	cleanedURL, error := url.Parse(url.String())
	if error != nil {
		return false, nil
	}

	harvestedParams := cleanedURL.Query()
	type ParamMatch struct {
		paramName string
		reason    string
	}
	var cleanedParams []ParamMatch
	for paramName := range harvestedParams {
		remove, reason := rule.RemoveQueryParamFromResourceURL(paramName)
		if remove {
			harvestedParams.Del(paramName)
			cleanedParams = append(cleanedParams, ParamMatch{paramName, reason})
		}
	}

	if len(cleanedParams) > 0 {
		cleanedURL.RawQuery = harvestedParams.Encode()
		return true, cleanedURL
	}
	return false, nil
}

// HarvestResource creates a Resource from a given URL and curation rules
func HarvestResource(origURLtext string, cleanCurationTargetRule CleanResourceParamsRule, ignoreCurationTargetRule IgnoreResourceRule,
	followHTMLRedirect FollowRedirectsInCurationTargetHTMLPayload) *Resource {
	result := new(Resource)
	result.origURLtext = origURLtext
	result.harvestedOn = time.Now()

	// Use the standard Go HTTP library method to retrieve the content; the
	// default will automatically follow redirects (e.g. HTTP redirects)
	resp, err := http.Get(origURLtext)
	result.isURLValid = err == nil
	if result.isURLValid == false {
		result.isDestValid = false
		result.isURLIgnored = true
		result.ignoreReason = fmt.Sprintf("Invalid URL '%s'", origURLtext)
		return result
	}

	result.httpStatusCode = resp.StatusCode
	if result.httpStatusCode != 200 {
		result.isDestValid = false
		result.isURLIgnored = true
		result.ignoreReason = fmt.Sprintf("Invalid HTTP Status Code %d", resp.StatusCode)
		return result
	}

	result.resolvedURL = resp.Request.URL
	result.finalURL = result.resolvedURL
	ignoreURL, ignoreReason := ignoreCurationTargetRule.IgnoreResource(result.resolvedURL)
	if ignoreURL {
		result.isDestValid = true
		result.isURLIgnored = true
		result.ignoreReason = ignoreReason
		return result
	}

	result.isURLIgnored = false
	result.isDestValid = true
	urlsParamsCleaned, cleanedURL := cleanResource(result.resolvedURL, cleanCurationTargetRule)
	if urlsParamsCleaned {
		result.cleanedURL = cleanedURL
		result.finalURL = cleanedURL
		result.isURLCleaned = true
	} else {
		result.isURLCleaned = false
	}

	result.inspectionResults = inspectContent(result.finalURL, resp)

	h := sha1.New()
	if result.isDestValid {
		h.Write([]byte(result.finalURL.String()))
	} else {
		h.Write([]byte(origURLtext))
	}
	bs := h.Sum(nil)
	result.globallyUniqueKey = fmt.Sprintf("%x", bs)

	// TODO once the URL is cleaned, double-check the cleaned URL to see if it's a valid destination; if not, revert to non-cleaned version
	// this could be done recursively here or by the outer function. This is necessary because "cleaning" a URL and removing params might
	// break it so we need to revert to original.

	if followHTMLRedirect {
		isHTMLRedirect, htmlRedirectURL := result.IsHTMLRedirect()
		if isHTMLRedirect {
			redirected := HarvestResource(htmlRedirectURL, cleanCurationTargetRule, ignoreCurationTargetRule, followHTMLRedirect)
			redirected.origResource = result
			return redirected
		}
	}

	return result
}

// HarvestResourceWithDefaults creates a Resource from a given URL using default rules
func HarvestResourceWithDefaults(origURLtext string) *Resource {
	return HarvestResource(origURLtext, defaultCleanURLsRegExList, defaultIgnoreURLsRegExList, true)
}
