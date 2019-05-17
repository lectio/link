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
func (r TraversedLink) OriginalURL() string {
	return r.OrigURLText
}

// FinalURL returns the fully resolved, "final" URL (after redirects, cleaning, ignoring, and all other rules are processed) or an error
func (r TraversedLink) FinalURL() (*url.URL, error) {
	return r.FinalizedURL, nil
}

// Ignore returns true if the URL should be ignored an a string for the reason
func (r TraversedLink) Ignore() (bool, string) {
	return r.IsURLIgnored, r.IgnoreReason
}

// IsHTMLRedirect returns true if redirect was requested through via <meta http-equiv='refresh' Content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (r *TraversedLink) IsHTMLRedirect() (bool, string) {
	if r.Content != nil {
		return r.Content.Redirect()
	}
	return false, ""
}
