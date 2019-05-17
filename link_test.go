package link

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/lectio/resource"
	"github.com/stretchr/testify/suite"
)

type LinkSuite struct {
	suite.Suite
	factory             *DefaultFactory
	followHTMLRedirects bool
}

func (suite *LinkSuite) SetupSuite() {
	suite.factory = NewFactory(suite)
}

func (suite *LinkSuite) TearDownSuite() {
}

func (suite *LinkSuite) DownloadContent(ctx context.Context, url *url.URL, resp *http.Response, typ resource.Type) (bool, resource.Attachment, error) {
	return resource.DownloadFile(ctx, suite.factory, url, resp, typ)
}

func (suite *LinkSuite) FollowRedirectsInHTMLContent(context.Context, *url.URL) bool {
	return suite.followHTMLRedirects
}

func (suite *LinkSuite) traverseSingleURLFromMockTweet(text string, urlText string) *TraversedLink {
	ctx := context.Background()
	suite.followHTMLRedirects = false // during testing, we don't want redirects
	_, link, _ := suite.factory.TraverseLink(ctx, urlText)
	return link.(*TraversedLink)
}

func (suite *LinkSuite) TestInvalidlyFormattedURLs() {
	hr := suite.traverseSingleURLFromMockTweet("Test an invalidly formatted URL %s in a mock tweet", "https://t")
	suite.False(hr.IsURLValid, "URL should have invalid format")
	suite.Nil(hr.Content, "No content should be available")

	finalURL, finalURLErr := hr.FinalURL()
	suite.Nil(finalURL, "Ensure FinalURL is nil")
	suite.Nil(finalURLErr, "Ensure error is nil")
}

func (suite *LinkSuite) TestInvalidDestinationURLs() {
	hr := suite.traverseSingleURLFromMockTweet("Test a validly formatted URL %s but with invalid destination in a mock tweet", "https://t.co/fDxPF")
	suite.False(hr.IsURLValid, "URL should be considered invalid")
	suite.Nil(hr.Content, "No content should be available")

	finalURL, finalURLErr := hr.FinalURL()
	suite.Nil(finalURL, "Ensure FinalURL is nil")
	suite.Nil(finalURLErr, "Ensure error is nil")
}

func (suite *LinkSuite) TestSimplifiedHostnames() {
	url, _ := url.Parse("https://www.netspective.com")
	suite.Equal("netspective.com", GetSimplifiedHostname(url))
	suite.Equal("netspective", GetSimplifiedHostnameWithoutTLD(url))
	url, _ = url.Parse("https://news.healthcareguys.com")
	suite.Equal("news.healthcareguys.com", GetSimplifiedHostname(url))
	suite.Equal("news.healthcareguys", GetSimplifiedHostnameWithoutTLD(url))
}

func (suite *LinkSuite) TestOpenGraphMetaTags() {
	hr := suite.traverseSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test01")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.NotNil(hr.Content, "Inspection results should be available")

	value, _, _ := hr.Content.MetaTag("og:site_name")
	suite.Equal(value, "Netspective")

	value, _, _ = hr.Content.MetaTag("og:title")
	suite.Equal(value, "Safety, privacy, and security focused technology consulting")

	value, _, _ = hr.Content.MetaTag("og:description")
	suite.Equal(value, "Software, technology, and management consulting focused on firms im pacted by FDA, ONC, NIST or other safety, privacy, and security regulations")
}

func (suite *LinkSuite) TestIgnoreRules() {
	hr := suite.traverseSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore", "https://t.co/xNzrxkHE1u")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsURLIgnored, "URL should be ignored (skipped)")
	suite.Equal(hr.IgnoreReason, "Matched Ignore Rule `^https://twitter.com/(.*?)/status/(.*)$`")

	finalURL, issue := hr.FinalURL()
	suite.Nil(issue, "Ensure there is no issue")
	suite.Equal("https://twitter.com/Live5News/status/993220120402161664/photo/1", finalURL.String())
}

func (suite *LinkSuite) TestResolvedURLRedirectedThroughHTMLProperly() {
	hr := suite.traverseSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to resolve via <meta http-equiv='refresh' content='delay;url='>, with utm_* params", "http://bit.ly/lectio_harvester_resource_test03")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	isHTMLRedirect, htmlRedirectURLText := hr.Content.Redirect()
	suite.True(isHTMLRedirect, "There should have been an HTML redirect requested through <meta http-equiv='refresh' content='delay;url='>")
	suite.Equal(htmlRedirectURLText, "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.NotNil(hr.Content, "Inspection results should be available")

	// at this point we want to get the "new" (redirected) and test it
	ctx := context.Background()
	suite.followHTMLRedirects = true
	traversable, redirLink, err := suite.factory.TraverseLink(ctx, "http://bit.ly/lectio_harvester_resource_test03")
	redirectedLink := redirLink.(*TraversedLink)
	suite.Nil(err, "No error expected")
	suite.True(traversable, "Should be traversable")
	suite.NotNil(redirectedLink.OrigLink, hr, "The referral resource should be the same as the original")
	suite.True(redirectedLink.IsURLValid, "Redirected URL should be formatted validly")
	suite.False(redirectedLink.IsURLIgnored, "Redirected URL should not be ignored")
	suite.True(redirectedLink.AreURLParamsCleaned, "Redirected URL should be 'cleaned'")
	suite.Equal(redirectedLink.ResolvedURL.String(), "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(redirectedLink.CleanedURL.String(), "https://www.netspective.com/")
	suite.Equal(redirectedLink.FinalizedURL.String(), redirectedLink.CleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(redirectedLink.Content, "Inspection results should be available")
}

func (suite *LinkSuite) TestResolvedURLCleaned() {
	hr := suite.traverseSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test01")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.True(hr.AreURLParamsCleaned, "URL should be 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(hr.CleanedURL.String(), "https://www.netspective.com/")
	suite.Equal(hr.FinalizedURL.String(), hr.CleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(hr.Content, "Inspection results should be available")
}

func (suite *LinkSuite) TestResolvedURLCleanedKeys() {
	hr := suite.traverseSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test02")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.True(hr.AreURLParamsCleaned, "URL should be 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "https://www.netspective.com/solutions/opsfolio/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(hr.CleanedURL.String(), "https://www.netspective.com/solutions/opsfolio/")
	suite.Equal(hr.FinalizedURL.String(), hr.CleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(hr.Content, "Inspection results should be available")
}

func (suite *LinkSuite) TestResolvedURLNotCleaned() {
	hr := suite.traverseSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore", "https://t.co/ELrZmo81wI")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.False(hr.AreURLParamsCleaned, "URL should not have been 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "https://www.foxnews.com/lifestyle/photo-of-donald-trump-look-alike-in-spain-goes-viral")
	suite.Equal(hr.FinalizedURL.String(), hr.ResolvedURL.String(), "finalURL should be same as resolvedURL")
	suite.Nil(hr.CleanedURL, "cleanedURL should be empty")

	content := hr.Content
	suite.NotNil(content, "The destination content should be available")
	suite.True(content.IsValid(), "The destination content should be valid")
	suite.True(content.IsHTML(), "The destination content should be HTML")
}

func (suite *LinkSuite) TestResolvedDocumentURLNotCleaned() {
	hr := suite.traverseSingleURLFromMockTweet("Check out the PROV-O specification document %s, which should resolve to an 'attachment' style URL", "http://ceur-ws.org/Vol-1401/paper-05.pdf")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.False(hr.AreURLParamsCleaned, "URL should not have been 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "http://ceur-ws.org/Vol-1401/paper-05.pdf")
	suite.Equal(hr.FinalizedURL.String(), hr.ResolvedURL.String(), "finalURL should be same as resolvedURL")
	suite.Nil(hr.CleanedURL, "cleanedURL should be empty")

	attachment := hr.Content.Attachment()
	suite.NotNil(attachment, "Should have an attachment")
	suite.True(attachment.IsValid(), "First attachment should be valid")
	suite.Equal("application/pdf", attachment.Type().ContentType(), "Should be a PDF file")
	suite.Equal("application/pdf", attachment.Type().MediaType(), "Should be a PDF file")

	fa, ok := attachment.(*resource.FileAttachment)
	suite.True(ok, "Attachment should be a FileAttachment type")
	if ok {
		fileExists := false
		if _, err := os.Stat(fa.DestPath); err == nil {
			fileExists = true
		}
		suite.True(fileExists, "File %s should exist", fa.DestPath)
		suite.Equal(path.Ext(fa.DestPath), ".pdf", "File's extension should be .pdf")

		fa.Delete()
		if _, err := os.Stat(fa.DestPath); err == nil {
			fileExists = true
		}
		suite.True(fileExists, "File %s should not exist", fa.DestPath)
	}
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(LinkSuite))
}
