package link

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Link is the public interface for a "smart URL" which knows its destination
type Link interface {
	OriginalURLText() string
	PrimaryKey(keys Keys) string
	FinalURL() (*url.URL, error)
	URLStructureValid() bool
	DestinationValid() bool
	Ignore() (bool, string)
}

// Lifecycle defines common creation / destruction methods
type Lifecycle interface {
	HarvestLink(urlText string) (Link, error)
}

// Reader defines common reader methods
type Reader interface {
	GetLink(urlText string) (Link, error)
	HasLink(urlText string) (bool, error)
}

// Writer defines common writer methods
type Writer interface {
	WriteLink(Link) error
	DeleteLink(Link) error
}

// Store pulls together all the lifecyle, reader, and writer methods
type Store interface {
	Reader
	Writer
	io.Closer
}

// HarvestedLink tracks a single URL that was curated or discovered in Content.
// Discovered URLs are validated, follow their redirects, and may have
// query parameters "cleaned" (if instructed).
type HarvestedLink struct {
	// TODO consider adding source information (e.g. tweet, e-mail, etc.) and embed style (e.g. text, HTML <a> tag, etc.)
	HarvestedOn         time.Time      `json:"harvestedOn,omitempty"`
	OrigURLText         string         `json:"origURLtext"`
	OrigLink            *HarvestedLink `json:"origLink,omitempty"`
	IsURLValid          bool           `json:"isURLStructureValid"`
	IsDestValid         bool           `json:"isDestinationValid"`
	HTTPStatusCode      int            `json:"httpStatusCode"`
	IsURLIgnored        bool           `json:"isURLIgnored"`
	IgnoreReason        string         `json:"ignoreReason"`
	AreURLParamsCleaned bool           `json:"areURLParamsCleaned"`
	ResolvedURL         *url.URL       `json:"resolvedURL"`
	CleanedURL          *url.URL       `json:"cleanedURL"`
	FinalizedURL        *url.URL       `json:"finalizedURL"`
	Content             *Content       `json:"content"`
}

// OriginalURLText returns the URL text that was parsed
func (r HarvestedLink) OriginalURLText() string {
	return r.OrigURLText
}

// FinalURL returns the fully resolved, "final" URL (after redirects, cleaning, ignoring, and all other rules are processed) or an error
func (r *HarvestedLink) FinalURL() (*url.URL, error) {
	if r.IsURLIgnored {
		return nil, fmt.Errorf("ignoring %q: %v", r.OrigURLText, r.IgnoreReason)
	}
	if !r.IsURLValid || !r.IsDestValid {
		return nil, fmt.Errorf("URL %q issue, IsURLValid: %v, IsDestValid: %v", r.OrigURLText, r.IsURLValid, r.IsDestValid)
	}
	if r.FinalizedURL == nil {
		return nil, fmt.Errorf("HarvestedLink %q FinalizedURL is nil", r.OrigURLText)
	}
	if len(r.FinalizedURL.String()) == 0 {
		return nil, fmt.Errorf("HarvestedLink %q FinalizedURL is empty string", r.OrigURLText)
	}
	return r.FinalizedURL, nil
}

// URLStructureValid returns true if the URL was properly parsed (does not indicate whether the destination is valid, though)
func (r HarvestedLink) URLStructureValid() bool {
	return r.IsURLValid
}

// DestinationValid returns true if the URL's format is valid and the destination was reachable
func (r HarvestedLink) DestinationValid() bool {
	return r.IsDestValid
}

// Ignore returns true if the URL should be ignored an a string for the reason
func (r HarvestedLink) Ignore() (bool, string) {
	return r.IsURLIgnored, r.IgnoreReason
}

// PrimaryKey returns the primary key for this URL
func (r HarvestedLink) PrimaryKey(keys Keys) string {
	if r.IsDestValid && r.FinalizedURL != nil {
		return keys.PrimaryKeyForURLText(r.FinalizedURL.String())
	} else {
		return keys.PrimaryKeyForURLText(r.OrigURLText)
	}
}

// IsHTMLRedirect returns true if redirect was requested through via <meta http-equiv='refresh' Content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (r *HarvestedLink) IsHTMLRedirect() (bool, string) {
	if r.Content != nil {
		return r.Content.IsContentBasedRedirect()
	}
	return false, ""
}

// cleanLink checks to see if there are any parameters that should be removed (e.g. UTM_*)
func cleanLink(url *url.URL, rule CleanLinkParamsRule) (bool, *url.URL) {
	if !rule.CleanLinkParams(url) {
		return false, nil
	}

	// make a copy because we're planning on changing the URL params
	CleanedURL, error := url.Parse(url.String())
	if error != nil {
		return false, nil
	}

	harvestedParams := CleanedURL.Query()
	type ParamMatch struct {
		paramName string
		reason    string
	}
	var cleanedParams []ParamMatch
	for paramName := range harvestedParams {
		remove, reason := rule.RemoveQueryParamFromLinkURL(url, paramName)
		if remove {
			harvestedParams.Del(paramName)
			cleanedParams = append(cleanedParams, ParamMatch{paramName, reason})
		}
	}

	if len(cleanedParams) > 0 {
		CleanedURL.RawQuery = harvestedParams.Encode()
		return true, CleanedURL
	}
	return false, nil
}

// HarvestLink creates a HarvestedLink from a given URL and curation rules
func HarvestLink(origURLtext string, cleanCurationTargetRule CleanLinkParamsRule, ignoreCurationTargetRule IgnoreLinkRule,
	destRule DestinationRule) *HarvestedLink {
	result := new(HarvestedLink)
	result.OrigURLText = origURLtext
	result.HarvestedOn = time.Now()

	// Use the standard Go HTTP library method to retrieve the Content; the
	// default will automatically follow redirects (e.g. HTTP redirects)
	// TODO: Consider using [HTTP Cache](https://github.com/gregjones/httpcache)
	resp, err := http.Get(origURLtext)
	result.IsURLValid = err == nil
	if result.IsURLValid == false {
		result.IsDestValid = false
		result.IsURLIgnored = true
		result.IgnoreReason = fmt.Sprintf("Invalid URL %q (%v)", origURLtext, err)
		return result
	}

	result.HTTPStatusCode = resp.StatusCode
	if result.HTTPStatusCode != 200 {
		result.IsDestValid = false
		result.IsURLIgnored = true
		result.IgnoreReason = fmt.Sprintf("Invalid HTTP Status Code %d", resp.StatusCode)
		return result
	}

	result.ResolvedURL = resp.Request.URL
	result.FinalizedURL = result.ResolvedURL
	ignoreURL, ignoreReason := ignoreCurationTargetRule.IgnoreLink(result.ResolvedURL)
	if ignoreURL {
		result.IsDestValid = true
		result.IsURLIgnored = true
		result.IgnoreReason = ignoreReason
		return result
	}

	result.IsURLIgnored = false
	result.IsDestValid = true
	urlsParamsCleaned, cleanedURL := cleanLink(result.ResolvedURL, cleanCurationTargetRule)
	if urlsParamsCleaned {
		result.CleanedURL = cleanedURL
		result.FinalizedURL = cleanedURL
		result.AreURLParamsCleaned = true
	} else {
		result.AreURLParamsCleaned = false
	}

	result.Content = MakeContent(result.FinalizedURL, resp, destRule)

	// TODO once the URL is cleaned, double-check the cleaned URL to see if it's a valid destination; if not, revert to non-cleaned version
	// this could be done recursively here or by the outer function. This is necessary because "cleaning" a URL and removing params might
	// break it so we need to revert to original.

	if destRule.FollowRedirectsInDestinationHTMLContent(result.FinalizedURL) {
		isHTMLRedirect, htmlRedirectURL := result.IsHTMLRedirect()
		if isHTMLRedirect {
			redirected := HarvestLink(htmlRedirectURL, cleanCurationTargetRule, ignoreCurationTargetRule, destRule)
			redirected.OrigLink = result
			return redirected
		}
	}

	return result
}

// HarvestLinkWithConfig creates a HarvestedLink from a given URL using configuration structure
func HarvestLinkWithConfig(OrigURLtext string, config *Configuration) *HarvestedLink {
	return HarvestLink(OrigURLtext, config, config, config)
}
