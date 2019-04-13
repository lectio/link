package link

import (
	"fmt"
	"net/url"
	"regexp"
)

// IgnoreLinkRule indicates whether a given URL should be ignored or harvested
type IgnoreLinkRule interface {
	IgnoreLink(url *url.URL) (bool, string)
}

// CleanLinkParamsRule indicates whether a specific URL parameter should be "cleaned" (removed)
type CleanLinkParamsRule interface {
	CleanLinkParams(url *url.URL) bool
	RemoveQueryParamFromLinkURL(url *url.URL, paramName string) (bool, string)
}

// DestinationRule indicates whether we want to perform any destination actions
type DestinationRule interface {
	FollowRedirectsInDestinationHTMLContent(url *url.URL) bool
	ParseMetaDataInDestinationHTMLContent(url *url.URL) bool
	DownloadAttachmentsFromDestination(url *url.URL) (bool, string)
}

// Configuration manage  the link harvesting options
type Configuration struct {
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
	result.IgnoreURLsRegExprs = []*regexp.Regexp{regexp.MustCompile(`^https://twitter.com/(.*?)/status/(.*)$`), regexp.MustCompile(`https://t.co`)}
	result.RemoveParamsFromURLsRegEx = []*regexp.Regexp{regexp.MustCompile(`^utm_`)}
	result.FollowHTMLRedirects = true
	result.ParseHTMLMetaDataTags = true
	result.DownloadLinkAttachments = false
	return result
}

// FollowRedirectsInDestinationHTMLContent defines whether we follow redirect rules in HTML <meta> refresh tags
func (c Configuration) FollowRedirectsInDestinationHTMLContent(url *url.URL) bool {
	return c.FollowHTMLRedirects
}

// ParseMetaDataInDestinationHTMLContent should be true if OpenGraph, TwitterCard, or other HTML meta data is required
func (c Configuration) ParseMetaDataInDestinationHTMLContent(url *url.URL) bool {
	return c.ParseHTMLMetaDataTags
}

// DownloadAttachmentsFromDestination defines whether we download link attachments
func (c Configuration) DownloadAttachmentsFromDestination(url *url.URL) (bool, string) {
	if c.DownloadLinkAttachments {
		return c.DownloadLinkAttachments, c.LinkAttachmentsStorePath
	}
	return false, c.LinkAttachmentsStorePath
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
