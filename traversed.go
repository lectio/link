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
	// TODO consider adding source information (e.g. tweet, e-mail, etc.) and embed style (e.g. text, HTML <a> tag, etc.)
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
	AllIssues           []Issue          `json:"issues"`
}

// OriginalURL returns the URL text that was parsed
func (r TraversedLink) OriginalURL() string {
	return r.OrigURLText
}

// Issues contains the problems in this link plus satisfies the Link interface
func (r TraversedLink) Issues() Issues {
	return r
}

// ErrorsAndWarnings contains the problems in this link plus satisfies the Link.Issues interface
func (r TraversedLink) ErrorsAndWarnings() []Issue {
	return r.AllIssues
}

// IssueCounts returns the total, errors, and warnings counts
func (r TraversedLink) IssueCounts() (uint, uint, uint) {
	if r.AllIssues == nil {
		return 0, 0, 0
	}
	var errors, warnings uint
	for _, i := range r.AllIssues {
		if i.IsError() {
			errors++
		} else {
			warnings++
		}
	}
	return uint(len(r.AllIssues)), errors, warnings
}

// HandleIssues loops through each issue and calls a particular handler
func (r TraversedLink) HandleIssues(errorHandler func(Issue), warningHandler func(Issue)) {
	if r.AllIssues == nil {
		return
	}
	for _, i := range r.AllIssues {
		if i.IsError() && errorHandler != nil {
			errorHandler(i)
		}
		if i.IsWarning() && warningHandler != nil {
			warningHandler(i)
		}
	}
}

// FinalURL returns the fully resolved, "final" URL (after redirects, cleaning, ignoring, and all other rules are processed) or an error
func (r *TraversedLink) FinalURL() (*url.URL, error) {
	return r.FinalizedURL, nil
}

// Ignore returns true if the URL should be ignored an a string for the reason
func (r TraversedLink) Ignore() (bool, string) {
	return r.IsURLIgnored, r.IgnoreReason
}

// PrimaryKey returns the primary key for this URL
func (r TraversedLink) PrimaryKey(keys Keys) string {
	if r.IsURLValid && r.FinalizedURL != nil {
		return keys.PrimaryKeyForURLText(r.FinalizedURL.String())
	}
	return keys.PrimaryKeyForURLText(r.OrigURLText)
}

// IsHTMLRedirect returns true if redirect was requested through via <meta http-equiv='refresh' Content='delay;url='>
// For an explanation, please see http://redirectdetective.com/redirection-types.html
func (r *TraversedLink) IsHTMLRedirect() (bool, string) {
	if r.Content != nil {
		return r.Content.Redirect()
	}
	return false, ""
}

// cleanLink checks to see if there are any parameters that should be removed (e.g. UTM_*)
func cleanLink(url *url.URL, rule CleanLinkQueryParamsPolicy) (bool, *url.URL) {
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

// TraverseLink creates a TraversedLink from a given URL and curation rules
func TraverseLink(origURLtext string, clqpp CleanLinkQueryParamsPolicy, ilp IgnoreLinkPolicy, dp DestinationPolicy) *TraversedLink {
	result := new(TraversedLink)
	result.OrigURLText = origURLtext
	result.TraversedOn = time.Now()

	var issue Issue
	result.Content, issue = resource.NewPageFromURL(origURLtext, dp)
	result.IsURLValid = issue == nil
	if result.IsURLValid == false {
		result.IsURLIgnored = true
		result.IgnoreReason = "Unable to construct URL"
		result.AllIssues = append(result.AllIssues, issue)
		return result
	}

	result.ResolvedURL = result.Content.URL()
	result.FinalizedURL = result.ResolvedURL
	ignoreURL, ignoreReason := ilp.IgnoreLink(result.ResolvedURL)
	if ignoreURL {
		result.IsURLIgnored = true
		result.IgnoreReason = ignoreReason
		result.AllIssues = append(result.AllIssues, newIssue(result.FinalizedURL.String(), MatchesIgnorePolicy, result.IgnoreReason, false))
		return result
	}

	result.IsURLIgnored = false
	urlsParamsCleaned, cleanedURL := cleanLink(result.ResolvedURL, clqpp)
	if urlsParamsCleaned {
		result.CleanedURL = cleanedURL
		result.FinalizedURL = cleanedURL
		result.AreURLParamsCleaned = true
	} else {
		result.AreURLParamsCleaned = false
	}

	// TODO once the URL is cleaned, double-check the cleaned URL to see if it's a valid destination; if not, revert to non-cleaned version
	// this could be done recursively here or by the outer function. This is necessary because "cleaning" a URL and removing params might
	// break it so we need to revert to original.

	if dp.FollowRedirectsInHTMLContent(result.FinalizedURL) {
		isHTMLRedirect, htmlRedirectURL := result.Content.Redirect()
		if isHTMLRedirect {
			redirected := TraverseLink(htmlRedirectURL, clqpp, ilp, dp)
			redirected.OrigLink = result
			return redirected
		}
	}

	return result
}

// TraverseLinkWithConfig creates a TraversedLink from a given URL using configuration structure
func TraverseLinkWithConfig(OrigURLtext string, config *Configuration) *TraversedLink {
	return TraverseLink(OrigURLtext, config, config, config)
}
