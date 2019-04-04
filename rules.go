package link

import (
	"fmt"
	"net/url"
	"regexp"
)

// IgnoreResourceRule is a rule
type IgnoreResourceRule interface {
	IgnoreResource(url *url.URL) (bool, string)
}

// CleanResourceParamsRule is a rule
type CleanResourceParamsRule interface {
	CleanResourceParams(url *url.URL) bool
	RemoveQueryParamFromResourceURL(paramName string) (bool, string)
}

// FollowRedirectsInCurationTargetHTMLPayload defines whether we follow redirect rules in HTML <meta> refresh tags
type FollowRedirectsInCurationTargetHTMLPayload bool

type ignoreURLsRegExList []*regexp.Regexp
type removeParamsFromURLsRegExList []*regexp.Regexp

var defaultIgnoreURLsRegExList ignoreURLsRegExList = []*regexp.Regexp{regexp.MustCompile(`^https://twitter.com/(.*?)/status/(.*)$`), regexp.MustCompile(`https://t.co`)}
var defaultCleanURLsRegExList removeParamsFromURLsRegExList = []*regexp.Regexp{regexp.MustCompile(`^utm_`)}
var defaultWebPrefixRegEx = regexp.MustCompile(`^www.`)                 // Removes "www." from start of source links
var defaultTopLevelDomainSuffixRegEx = regexp.MustCompile(`\.[^\.]+?$`) // Removes ".com" and other TLD suffixes from end of hostname

func (l ignoreURLsRegExList) IgnoreResource(url *url.URL) (bool, string) {
	URLtext := url.String()
	for _, regEx := range l {
		if regEx.MatchString(URLtext) {
			return true, fmt.Sprintf("Matched Ignore Rule `%s`", regEx.String())
		}
	}
	return false, ""
}

func (l removeParamsFromURLsRegExList) CleanResourceParams(url *url.URL) bool {
	// we try to clean all URLs, not specific ones
	return true
}

func (l removeParamsFromURLsRegExList) RemoveQueryParamFromResourceURL(paramName string) (bool, string) {
	for _, regEx := range l {
		if regEx.MatchString(paramName) {
			return true, fmt.Sprintf("Matched cleaner rule `%s`", regEx.String())
		}
	}

	return false, ""
}

// GetSimplifiedHostname returns the URL's hostname without 'www.' prefix
func GetSimplifiedHostname(url *url.URL) string {
	return defaultWebPrefixRegEx.ReplaceAllString(url.Hostname(), "")
}

// GetSimplifiedHostnameWithoutTLD returns the URL's hostname without 'www.' prefix and removes the top level domain suffix (.com, etc.)
func GetSimplifiedHostnameWithoutTLD(url *url.URL) string {
	simplified := GetSimplifiedHostname(url)
	return defaultTopLevelDomainSuffixRegEx.ReplaceAllString(simplified, "")
}
