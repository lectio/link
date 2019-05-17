package link

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/lectio/resource"
	"golang.org/x/xerrors"
	"net/url"
	"os"
	"path"
	"regexp"
	"time"
)

// Link is the public interface for a "smart URL" which knows its destination
type Link interface {
	OriginalURL() string
	FinalURL() (*url.URL, error)
}

// Factory is a lifecycle manager for URL-based resources
type Factory interface {
	TraverseLink(ctx context.Context, origURLtext string, options ...interface{}) (bool, Link, error)
}

// NewFactory creates a new thread-safe resource factory
func NewFactory(options ...interface{}) *DefaultFactory {
	f := &DefaultFactory{}

	f.ResourceFactory = resource.NewFactory(options...)
	f.WarningTracker = f // we implemented a default version

	f.IgnoreLinkPolicy = f // we implemented a default version
	f.IgnoreURLsRegExprs = []*regexp.Regexp{regexp.MustCompile(`^https://twitter.com/(.*?)/status/(.*)$`), regexp.MustCompile(`https://t.co`)}
	f.RemoveParamsFromURLsRegEx = []*regexp.Regexp{regexp.MustCompile(`^utm_`)}

	f.CleanLinkQueryParamsPolicy = f         // we implemented a default version
	f.FollowRedirectsInHTMLContentPolicy = f // we implemented a default version

	f.DownloadAttachmentsStoragePolicy = f

	f.initOptions(options...)

	return f
}

// IgnoreLinkPolicy indicates whether a given URL should be ignored or harvested
type IgnoreLinkPolicy interface {
	IgnoreLink(context.Context, *url.URL) (bool, string)
}

// CleanLinkQueryParamsPolicy indicates whether a specific URL parameter should be "cleaned" (removed)
type CleanLinkQueryParamsPolicy interface {
	CleanLinkParams(ctx context.Context, url *url.URL) bool
	RemoveQueryParamFromLinkURL(ctx context.Context, url *url.URL, paramName string) (bool, string)
}

// FollowRedirectsInHTMLContentPolicy indicates whether we want to perform any destination actions
type FollowRedirectsInHTMLContentPolicy interface {
	FollowRedirectsInHTMLContent(context.Context, *url.URL) bool
}

type FileStoragePolicy interface {
	FileStoragePath(context.Context) string
	FileName(ctx context.Context, suggested string) string
}

type WarningTracker interface {
	OnWarning(ctx context.Context, code, message string)
}

type DefaultFactory struct {
	IgnoreURLsRegExprs        []*regexp.Regexp `json:"ignoreURLsRegExprs"`
	RemoveParamsFromURLsRegEx []*regexp.Regexp `json:"removeParamsFromURLsRegEx"`

	ResourceFactory                    resource.Factory
	WarningTracker                     WarningTracker
	IgnoreLinkPolicy                   IgnoreLinkPolicy
	CleanLinkQueryParamsPolicy         CleanLinkQueryParamsPolicy
	FollowRedirectsInHTMLContentPolicy FollowRedirectsInHTMLContentPolicy
	DownloadAttachmentsStoragePolicy   FileStoragePolicy
}

func (f *DefaultFactory) initOptions(options ...interface{}) {
	for _, option := range options {
		if wt, ok := option.(WarningTracker); ok {
			f.WarningTracker = wt
		}
		if instance, ok := option.(IgnoreLinkPolicy); ok {
			f.IgnoreLinkPolicy = instance
		}
		if instance, ok := option.(CleanLinkQueryParamsPolicy); ok {
			f.CleanLinkQueryParamsPolicy = instance
		}
		if instance, ok := option.(FollowRedirectsInHTMLContentPolicy); ok {
			f.FollowRedirectsInHTMLContentPolicy = instance
		}
		if instance, ok := option.(FileStoragePolicy); ok {
			f.DownloadAttachmentsStoragePolicy = instance
		}
	}
}

// FollowRedirectsInHTMLContent is the default implementation
func (f *DefaultFactory) FollowRedirectsInHTMLContent(context.Context, *url.URL) bool {
	return true
}

// IgnoreLink returns true (and a reason) if the given url should be ignored by the harvester
func (f *DefaultFactory) IgnoreLink(ctx context.Context, url *url.URL) (bool, string) {
	URLtext := url.String()
	for _, regEx := range f.IgnoreURLsRegExprs {
		if regEx.MatchString(URLtext) {
			return true, fmt.Sprintf("Matched Ignore Rule `%s`", regEx.String())
		}
	}
	return false, ""
}

// CleanLinkParams returns true if the given url's query string param should be "cleaned" by the harvester
func (f *DefaultFactory) CleanLinkParams(ctx context.Context, url *url.URL) bool {
	// we try to clean all URLs, not specific ones
	return true
}

// RemoveQueryParamFromLinkURL returns true (and a reason) if the given url's specific query string param should be "cleaned" by the harvester
func (f *DefaultFactory) RemoveQueryParamFromLinkURL(ctx context.Context, url *url.URL, paramName string) (bool, string) {
	for _, regEx := range f.RemoveParamsFromURLsRegEx {
		if regEx.MatchString(paramName) {
			return true, fmt.Sprintf("Matched cleaner rule %q: %q", regEx.String(), url.String())
		}
	}

	return false, ""
}

// OnWarning is the default function if nothing else is provided in initOptions()
func (f *DefaultFactory) OnWarning(ctx context.Context, code string, message string) {
}

// CreateFile satisfies resource.FileAttachmentCreator method
func (f *DefaultFactory) CreateFile(ctx context.Context, url *url.URL, t resource.Type) (*os.File, error) {
	pathAndFileName := path.Join(f.FileStoragePath(ctx), f.FileName(ctx, url.String()))
	destFile, err := os.Create(pathAndFileName)
	if err != nil {
		xerrors.Errorf("Unable to create file %q: %w", pathAndFileName, err)
	}
	return destFile, nil
}

// AutoAssignExtension satisfies resource.FileAttachmentCreator method
func (f *DefaultFactory) AutoAssignExtension(context.Context, *url.URL, resource.Type) bool {
	return true
}

func (f *DefaultFactory) FileStoragePath(context.Context) string {
	return "" // current directory?
}

func (f *DefaultFactory) FileName(ctx context.Context, suggested string) string {
	h := sha1.New()
	h.Write([]byte(suggested))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

// TraverseLink creates a content instance from the given URL
func (f *DefaultFactory) TraverseLink(ctx context.Context, origURLtext string, options ...interface{}) (bool, Link, error) {
	result := new(TraversedLink)
	result.OrigURLText = origURLtext
	result.TraversedOn = time.Now()

	var err error
	result.Content, err = f.ResourceFactory.PageFromURL(ctx, origURLtext, options...)
	result.IsURLValid = err == nil
	if result.IsURLValid == false {
		result.IsURLIgnored = true
		result.IgnoreReason = "Unable to construct URL"
		return false, result, xerrors.Errorf("Unable to create page from URL: %w", err)
	}

	result.ResolvedURL = result.Content.URL()
	result.FinalizedURL = result.ResolvedURL
	ignoreURL, ignoreReason := f.IgnoreLinkPolicy.IgnoreLink(ctx, result.ResolvedURL)
	if ignoreURL {
		result.IsURLIgnored = true
		result.IgnoreReason = ignoreReason
		return false, result, nil
	}

	result.IsURLIgnored = false
	urlsParamsCleaned, cleanedURL := f.cleanLink(ctx, result.ResolvedURL)
	if urlsParamsCleaned {
		result.CleanedURL = cleanedURL
		result.FinalizedURL = cleanedURL
		result.AreURLParamsCleaned = true
	} else {
		result.AreURLParamsCleaned = false
	}

	// TODO: once the URL is cleaned, double-check the cleaned URL to see if it's a valid destination; if not, revert to non-cleaned version
	// this could be done recursively here or by the outer function. This is necessary because "cleaning" a URL and removing params might
	// break it so we need to revert to original.

	if f.FollowRedirectsInHTMLContentPolicy.FollowRedirectsInHTMLContent(ctx, result.FinalizedURL) {
		isHTMLRedirect, htmlRedirectURL := result.Content.Redirect()
		if isHTMLRedirect {
			traversable, redirLink, redirErr := f.TraverseLink(ctx, htmlRedirectURL, options...)
			redirected := redirLink.(*TraversedLink)
			redirected.OrigLink = result
			return traversable, redirected, redirErr
		}
	}

	return true, result, nil
}

// cleanLink checks to see if there are any parameters that should be removed (e.g. UTM_*)
func (f *DefaultFactory) cleanLink(ctx context.Context, url *url.URL) (bool, *url.URL) {
	if !f.CleanLinkQueryParamsPolicy.CleanLinkParams(ctx, url) {
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
		remove, reason := f.CleanLinkQueryParamsPolicy.RemoveQueryParamFromLinkURL(ctx, url, paramName)
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
