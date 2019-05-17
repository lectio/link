package link

import (
	"net/url"
	"regexp"
)

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
