package link

import (
	"github.com/lectio/resource"
	"net/url"
	"time"
)

// TraversedLink tracks a single URL that was curated or discovered in Content.
// Discovered URLs are validated, follow their redirects, and may have
// query parameters "cleaned" (if instructed).
type TraversedLink struct {
	TraversedOn         time.Time        `json:"traversedOn,omitempty"`
	OrigURLText         string           `json:"origURLtext"`
	OrigLink            *TraversedLink   `json:"origLink,omitempty"`
	IsURLValid          bool             `json:"isURLValid"`
	IsURLIgnored        bool             `json:"isURLIgnored"`
	IgnoreReason        string           `json:"ignoreReason"`
	AreURLParamsCleaned bool             `json:"areURLParamsCleaned"`
	ResolvedURL         *url.URL         `json:"resolvedURL"`
	CleanedURL          *url.URL         `json:"cleanedURL"`
	FinalizedURL        *url.URL         `json:"finalizedURL"`
	Content             resource.Content `json:"content"`
}

// OriginalURL returns the URL text that was parsed
func (l *TraversedLink) OriginalURL() string {
	return l.OrigURLText
}

// FinalURL returns the fully resolved, "final" URL (after redirects, cleaning, ignoring, and all other rules are processed) or an error
func (l *TraversedLink) FinalURL() (*url.URL, error) {
	return l.FinalizedURL, nil
}

// Ignore returns true if the URL should be ignored an a string for the reason
func (l *TraversedLink) Ignore() (bool, string) {
	return l.IsURLIgnored, l.IgnoreReason
}

// IsHTMLRedirect returns true if redirect was requested through via <meta http-equiv='refresh' Content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (l *TraversedLink) IsHTMLRedirect() (bool, string) {
	if l.Content != nil {
		return l.Content.Redirect()
	}
	return false, ""
}

// Traversable returns true if this link is traversable or has been traversed
func (l *TraversedLink) Traversable(warn func(code, message string)) bool {
	if !l.IsURLValid {
		warn("LECTIOLINK-001-INVALIDURL", l.IgnoreReason)
		return false
	}

	if l.IsURLIgnored {
		warn("LECTIOLINK-002-URLIGNORED", l.IgnoreReason)
		return false
	}

	return true
}
