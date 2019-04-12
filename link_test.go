package link

import (
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LinkSuite struct {
	suite.Suite
}

func (suite *LinkSuite) SetupSuite() {
}

func (suite *LinkSuite) TearDownSuite() {
}

func (suite *LinkSuite) harvestSingleURLFromMockTweet(text string, urlText string) *Link {
	config := MakeConfiguration()
	config.DownloadLinkAttachments = true
	// false for followHTMLRedirect because we need to test the features in the suite; in production it would be true
	config.FollowHTMLRedirects = false
	hr := HarvestLinkWithConfig(urlText, config)
	suite.NotNil(hr, "The harvested resources should not be Nil")
	return hr
}

func (suite *LinkSuite) TestInvalidlyFormattedURLs() {
	hr := suite.harvestSingleURLFromMockTweet("Test an invalidly formatted URL %s in a mock tweet", "https://t")
	suite.False(hr.IsURLValid, "URL should have invalid format")
	suite.False(hr.IsDestValid, "URL should have invalid destination")
	suite.Nil(hr.Content, "No content should be available")
}

func (suite *LinkSuite) TestInvalidDestinationURLs() {
	hr := suite.harvestSingleURLFromMockTweet("Test a validly formatted URL %s but with invalid destination in a mock tweet", "https://t.co/fDxPF")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.False(hr.IsDestValid, "URL should have invalid destination")
	suite.Equal(hr.HTTPStatusCode, 404)
	suite.Nil(hr.Content, "No content should be available")
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
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test01")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsDestValid, "URL should have valid destination")
	suite.NotNil(hr.Content, "Inspection results should be available")

	value, _ := hr.Content.GetOpenGraphMetaTag("site_name")
	suite.Equal(value, "Netspective")

	value, _ = hr.Content.GetOpenGraphMetaTag("title")
	suite.Equal(value, "Safety, privacy, and security focused technology consulting")

	value, _ = hr.Content.GetOpenGraphMetaTag("description")
	suite.Equal(value, "Software, technology, and management consulting focused on firms im pacted by FDA, ONC, NIST or other safety, privacy, and security regulations")
}

func (suite *LinkSuite) TestIgnoreRules() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore", "https://t.co/xNzrxkHE1u")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsDestValid, "URL should have valid destination")
	suite.True(hr.IsURLIgnored, "URL should be ignored (skipped)")
	suite.Equal(hr.IgnoreReason, "Matched Ignore Rule `^https://twitter.com/(.*?)/status/(.*)$`")
	suite.Nil(hr.Content, "No content should be available")
}

func (suite *LinkSuite) TestResolvedURLRedirectedThroughHTMLProperly() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to resolve via <meta http-equiv='refresh' content='delay;url='>, with utm_* params", "http://bit.ly/lectio_harvester_resource_test03")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsDestValid, "URL should have valid destination")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	isHTMLRedirect, htmlRedirectURLText := hr.Content.IsContentBasedRedirect()
	suite.True(isHTMLRedirect, "There should have been an HTML redirect requested through <meta http-equiv='refresh' content='delay;url='>")
	suite.Equal(htmlRedirectURLText, "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.NotNil(hr.Content, "Inspection results should be available")

	// at this point we want to get the "new" (redirected) and test it
	config := MakeConfiguration()
	config.FollowHTMLRedirects = true
	config.DownloadLinkAttachments = true
	redirectedHR := HarvestLinkWithConfig("http://bit.ly/lectio_harvester_resource_test03", config)
	suite.NotNil(redirectedHR.OrigLink, hr, "The referral resource should be the same as the original")
	suite.True(redirectedHR.IsURLValid, "Redirected URL should be formatted validly")
	suite.True(redirectedHR.IsDestValid, "Redirected URL should have valid destination")
	suite.False(redirectedHR.IsURLIgnored, "Redirected URL should not be ignored")
	suite.True(redirectedHR.AreURLParamsCleaned, "Redirected URL should be 'cleaned'")
	suite.Equal(redirectedHR.ResolvedURL.String(), "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(redirectedHR.CleanedURL.String(), "https://www.netspective.com/")
	suite.Equal(redirectedHR.FinalizedURL.String(), redirectedHR.CleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(redirectedHR.Content, "Inspection results should be available")
}

func (suite *LinkSuite) TestResolvedURLCleaned() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test01")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsDestValid, "URL should have valid destination")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.True(hr.AreURLParamsCleaned, "URL should be 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(hr.CleanedURL.String(), "https://www.netspective.com/")
	suite.Equal(hr.FinalizedURL.String(), hr.CleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(hr.Content, "Inspection results should be available")
}

func (suite *LinkSuite) TestResolvedURLCleanedKeys() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test02")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsDestValid, "URL should have valid destination")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.True(hr.AreURLParamsCleaned, "URL should be 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "https://www.netspective.com/solutions/opsfolio/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(hr.CleanedURL.String(), "https://www.netspective.com/solutions/opsfolio/")
	suite.Equal(hr.FinalizedURL.String(), hr.CleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(hr.Content, "Inspection results should be available")
	suite.Equal(hr.GloballyUniqueKey, "c3ac941bc19188497805cbe583ff8d122ac663d6")
}

func (suite *LinkSuite) TestResolvedURLNotCleaned() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore", "https://t.co/ELrZmo81wI")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsDestValid, "URL should have valid destination")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.False(hr.AreURLParamsCleaned, "URL should not have been 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "https://www.foxnews.com/lifestyle/photo-of-donald-trump-look-alike-in-spain-goes-viral")
	suite.Equal(hr.FinalizedURL.String(), hr.ResolvedURL.String(), "finalURL should be same as resolvedURL")
	suite.Nil(hr.CleanedURL, "cleanedURL should be empty")

	content := hr.Content
	suite.NotNil(content, "The destination content should be available")
	suite.True(content.IsValid(), "The destination content should be valid")
	suite.True(content.IsHTML(), "The destination content should be HTML")
	suite.False(content.WasDownloaded(), "Because the destination was HTML, it should not have required to be downloaded")
}

func (suite *LinkSuite) TestResolvedDocumentURLNotCleaned() {
	hr := suite.harvestSingleURLFromMockTweet("Check out the PROV-O specification document %s, which should resolve to an 'attachment' style URL", "http://ceur-ws.org/Vol-1401/paper-05.pdf")
	suite.True(hr.IsURLValid, "URL should be formatted validly")
	suite.True(hr.IsDestValid, "URL should have valid destination")
	suite.False(hr.IsURLIgnored, "URL should not be ignored")
	suite.False(hr.AreURLParamsCleaned, "URL should not have been 'cleaned'")
	suite.Equal(hr.ResolvedURL.String(), "http://ceur-ws.org/Vol-1401/paper-05.pdf")
	suite.Equal(hr.FinalizedURL.String(), hr.ResolvedURL.String(), "finalURL should be same as resolvedURL")
	suite.Nil(hr.CleanedURL, "cleanedURL should be empty")

	content := hr.Content
	suite.NotNil(content, "The destination content should be available")
	suite.True(content.IsValid(), "The destination content should be valid")
	suite.True(content.WasDownloaded(), "Because the destination wasn't HTML, it should have been downloaded")
	suite.Equal(content.Attachment.FileType.Extension, "pdf")

	fileExists := false
	if _, err := os.Stat(content.Attachment.DestPath); err == nil {
		fileExists = true
	}
	suite.True(fileExists, "File %s should exist", content.Attachment.DestPath)
	suite.Equal(path.Ext(content.Attachment.DestPath), ".pdf", "File's extension should be .pdf")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(LinkSuite))
}
