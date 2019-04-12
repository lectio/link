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

// Attachment manages any content that was downloaded for further inspection
type Attachment struct {
	url           *url.URL
	destPath      string
	downloadError error
	fileTypeError error
	fileType      types.Type
}

// IsValid returns true if there are no errors
func (a Attachment) IsValid() bool {
	if a.downloadError != nil {
		return false
	}
	if a.fileTypeError != nil {
		return false
	}

	return true
}

// Delete removes the file that was downloaded
func (a *Attachment) Delete() {
	os.Remove(a.destPath)
}

// download will download the URL as an "attachment" to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func downloadFile(url *url.URL, resp *http.Response, destFile *os.File) *Attachment {
	result := new(Attachment)
	result.url = url

	defer destFile.Close()
	defer resp.Body.Close()
	result.destPath = destFile.Name()
	_, err := io.Copy(destFile, resp.Body)
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

// downloadTemp will download the URL as an "attachment" to a temporary file.
func downloadTemp(url *url.URL, resp *http.Response, tempPattern string) *Attachment {
	destFile, err := ioutil.TempFile(os.TempDir(), tempPattern)

	if err != nil {
		result := new(Attachment)
		result.url = url
		result.downloadError = err
		return result
	}

	return downloadFile(url, resp, destFile)
}

// download will download the URL as an "attachment" to named file.
func download(url *url.URL, resp *http.Response, pathAndFileName string) *Attachment {
	destFile, err := os.Create(pathAndFileName)

	if err != nil {
		result := new(Attachment)
		result.url = url
		result.downloadError = err
		return result
	}

	return downloadFile(url, resp, destFile)
}

// InspectedContent manages the kind of content was inspected
type InspectedContent struct {
	url                          *url.URL
	contentType                  string
	mediaType                    string
	mediaTypeParams              map[string]string
	mediaTypeError               error
	htmlParsed                   bool
	htmlParseError               error
	isHTMLRedirect               bool
	metaRefreshTagContentURLText string            // if IsHTMLRedirect is true, then this is the value after url= in something like <meta http-equiv='refresh' content='delay;url='>
	metaPropertyTags             map[string]string // if IsHTML() is true, a collection of all meta data like <meta property="og:site_name" content="Netspective" /> or <meta name="twitter:title" content="text" />
	attachment                   *Attachment
}

// InspectCurationTarget will figure out what kind of destination content we're dealing with
func inspectContent(url *url.URL, resp *http.Response, destRule DestinationRule) *InspectedContent {
	result := new(InspectedContent)
	result.metaPropertyTags = make(map[string]string)
	result.url = url
	result.contentType = resp.Header.Get("Content-Type")
	if len(result.contentType) > 0 {
		result.mediaType, result.mediaTypeParams, result.mediaTypeError = mime.ParseMediaType(result.contentType)
		if result.mediaTypeError != nil {
			return result
		}
		if result.IsHTML() && (destRule.FollowRedirectsInDestinationHTMLContent(url) || destRule.ParseMetaDataInDestinationHTMLContent(url)) {
			result.parsePageMetaData(url, resp)
			result.htmlParsed = true
			return result
		}
	}

	// If we get to here it means that we need to download the content to inspect it any further.
	// We download it first because it's possible we want to retain it for later use.
	downloadAttachment, destFileName := destRule.DownloadAttachmentsFromDestination(url)
	if downloadAttachment {
		if len(destFileName) == 0 {
			result.attachment = downloadTemp(url, resp, "link-attachment-")
		} else {
			result.attachment = download(url, resp, destFileName)
		}
	}
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

// IsValid returns true if there are no errors
func (c InspectedContent) IsValid() bool {
	if c.mediaTypeError != nil {
		return false
	}

	if c.attachment != nil {
		return c.attachment.IsValid()
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

// IsHTMLRedirect returns true if redirect was requested through via <meta http-equiv='refresh' content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (c InspectedContent) IsHTMLRedirect() (bool, string) {
	return c.isHTMLRedirect, c.metaRefreshTagContentURLText
}

// WasDownloaded returns true if content was downloaded
func (c InspectedContent) WasDownloaded() bool {
	return c.attachment != nil
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

// FinalURL returns the fully resolved, "final" URL (after redirects and all other rules are processed) or an error
func (r *Resource) FinalURL() (*url.URL, error) {
	isIgnored, ignoreReason := r.IsIgnored()
	if isIgnored {
		return nil, fmt.Errorf("ignoring %q: %v", r.OriginalURLText(), ignoreReason)
	}
	isURLValid, isDestValid := r.IsValid()
	if !isURLValid || !isDestValid {
		return nil, fmt.Errorf("URL %q issue, isURLValid: %v, isDestValid: %v", r.OriginalURLText(), isURLValid, isDestValid)
	}
	if r.finalURL == nil {
		return nil, fmt.Errorf("resource %q finalURL is nil", r.OriginalURLText())
	}
	if len(r.finalURL.String()) == 0 {
		return nil, fmt.Errorf("resource %q finalURL is empty string", r.OriginalURLText())
	}
	return r.finalURL, nil
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
	destRule DestinationRule) *Resource {
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
		result.ignoreReason = fmt.Sprintf("Invalid URL %q (%v)", origURLtext, err)
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

	h := sha1.New()
	if result.isDestValid {
		h.Write([]byte(result.finalURL.String()))
	} else {
		h.Write([]byte(origURLtext))
	}
	bs := h.Sum(nil)
	result.globallyUniqueKey = fmt.Sprintf("%x", bs)

	result.inspectionResults = inspectContent(result.finalURL, resp, destRule)

	// TODO once the URL is cleaned, double-check the cleaned URL to see if it's a valid destination; if not, revert to non-cleaned version
	// this could be done recursively here or by the outer function. This is necessary because "cleaning" a URL and removing params might
	// break it so we need to revert to original.

	if destRule.FollowRedirectsInDestinationHTMLContent(result.finalURL) {
		isHTMLRedirect, htmlRedirectURL := result.IsHTMLRedirect()
		if isHTMLRedirect {
			redirected := HarvestResource(htmlRedirectURL, cleanCurationTargetRule, ignoreCurationTargetRule, destRule)
			redirected.origResource = result
			return redirected
		}
	}

	return result
}

// HarvestResourceWithConfig creates a Resource from a given URL using configuration structure
func HarvestResourceWithConfig(origURLtext string, config *Configuration) *Resource {
	return HarvestResource(origURLtext, config, config, config)
}
