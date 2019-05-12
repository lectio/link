package link

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"

	"github.com/lectio/resource"
)

// IgnoreLinkPolicy indicates whether a given URL should be ignored or harvested
type IgnoreLinkPolicy interface {
	IgnoreLink(url *url.URL) (bool, string)
}

// CleanLinkQueryParamsPolicy indicates whether a specific URL parameter should be "cleaned" (removed)
type CleanLinkQueryParamsPolicy interface {
	CleanLinkParams(url *url.URL) bool
	RemoveQueryParamFromLinkURL(url *url.URL, paramName string) (bool, string)
}

// DestinationPolicy indicates whether we want to perform any destination actions
type DestinationPolicy interface {
	resource.Policy
	FollowRedirectsInHTMLContent(url *url.URL) bool
}

// Configuration manages the link traversal options
type Configuration struct {
	httpClient                *http.Client
	IgnoreURLsRegExprs        []*regexp.Regexp `json:"ignoreURLsRegExprs"`
	RemoveParamsFromURLsRegEx []*regexp.Regexp `json:"removeParamsFromURLsRegEx"`
	FollowHTMLRedirects       bool             `json:"followHTMLRedirects"`
	ParseHTMLMetaDataTags     bool             `json:"parseHTMLMetaDataTags"`
	DownloadLinkAttachments   bool             `json:"downloadLinkAttachments"`
	LinkAttachmentsStorePath  string           `json:"linkAttachmentsStoragePath"`
}

// MakeConfiguration creates a default configuration instance
func MakeConfiguration() *Configuration {
	result := new(Configuration)
	result.httpClient = &http.Client{Timeout: resource.HTTPTimeout}
	result.IgnoreURLsRegExprs = []*regexp.Regexp{regexp.MustCompile(`^https://twitter.com/(.*?)/status/(.*)$`), regexp.MustCompile(`https://t.co`)}
	result.RemoveParamsFromURLsRegEx = []*regexp.Regexp{regexp.MustCompile(`^utm_`)}
	result.FollowHTMLRedirects = true
	result.ParseHTMLMetaDataTags = true
	result.DownloadLinkAttachments = false
	return result
}

// HTTPClient defines the HTTP Client for the links to use
// This method satisfies resource.Policy interface
func (c Configuration) HTTPClient() *http.Client {
	return c.httpClient
}

// PrepareRequest adjusts the user agent and other HTTP request settings
// This method satisfies resource.Policy interface
func (c Configuration) PrepareRequest(client *http.Client, req *http.Request) {
	req.Header.Set("User-Agent", "github.com/lectio/link")
}

// DetectRedirectsInHTMLContent defines whether we detect redirect rules in HTML <meta> refresh tags
// This method satisfies resource.Policy interface
func (c Configuration) DetectRedirectsInHTMLContent(*url.URL) bool {
	return c.FollowHTMLRedirects
}

// FollowRedirectsInHTMLContent defines whether we follow redirect rules in HTML <meta> refresh tags
func (c Configuration) FollowRedirectsInHTMLContent(url *url.URL) bool {
	return c.FollowHTMLRedirects
}

// ParseMetaDataInHTMLContent defines whether we want to parse HTML meta data
// This method satisfies resource.Policy interface
func (c Configuration) ParseMetaDataInHTMLContent(*url.URL) bool {
	return c.ParseHTMLMetaDataTags
}

// DownloadContent satisfies Policy method
func (c Configuration) DownloadContent(url *url.URL, resp *http.Response, typ resource.Type) (bool, resource.Attachment, []resource.Issue) {
	return resource.DownloadFile(c, url, resp, typ)
}

// CreateFile satisfies FileAttachmentPolicy method
func (c Configuration) CreateFile(url *url.URL, t resource.Type) (*os.File, resource.Issue) {
	pathAndFileName := path.Join(c.LinkAttachmentsStorePath, HashText(url.String()))
	var issue Issue
	destFile, err := os.Create(pathAndFileName)
	if err != nil {
		issue = newIssue(url.String(), "SUITE_E-0001", fmt.Sprintf("Unable to create file %q", pathAndFileName), true)
	}
	return destFile, issue
}

// AutoAssignExtension satisfies FileAttachmentPolicy method
func (c Configuration) AutoAssignExtension(url *url.URL, t resource.Type) bool {
	return true
}

// IgnoreLink returns true (and a reason) if the given url should be ignored by the harvester
func (c Configuration) IgnoreLink(url *url.URL) (bool, string) {
	URLtext := url.String()
	for _, regEx := range c.IgnoreURLsRegExprs {
		if regEx.MatchString(URLtext) {
			return true, fmt.Sprintf("Matched Ignore Rule `%s`", regEx.String())
		}
	}
	return false, ""
}

// CleanLinkParams returns true if the given url's query string param should be "cleaned" by the harvester
func (c Configuration) CleanLinkParams(url *url.URL) bool {
	// we try to clean all URLs, not specific ones
	return true
}

// RemoveQueryParamFromLinkURL returns true (and a reason) if the given url's specific query string param should be "cleaned" by the harvester
func (c Configuration) RemoveQueryParamFromLinkURL(url *url.URL, paramName string) (bool, string) {
	for _, regEx := range c.RemoveParamsFromURLsRegEx {
		if regEx.MatchString(paramName) {
			return true, fmt.Sprintf("Matched cleaner rule %q: %q", regEx.String(), url.String())
		}
	}

	return false, ""
}

var defaultWebPrefixRegEx = regexp.MustCompile(`^www.`)                 // Removes "www." from start of source links
var defaultTopLevelDomainSuffixRegEx = regexp.MustCompile(`\.[^\.]+?$`) // Removes ".com" and other TLD suffixes from end of hostname

// GetSimplifiedHostname returns the URL's hostname without 'www.' prefix
func GetSimplifiedHostname(url *url.URL) string {
	return defaultWebPrefixRegEx.ReplaceAllString(url.Hostname(), "")
}

// GetSimplifiedHostnameWithoutTLD returns the URL's hostname without 'www.' prefix and removes the top level domain suffix (.com, etc.)
func GetSimplifiedHostnameWithoutTLD(url *url.URL) string {
	simplified := GetSimplifiedHostname(url)
	return defaultTopLevelDomainSuffixRegEx.ReplaceAllString(simplified, "")
}
