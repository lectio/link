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

func (suite *LinkSuite) harvestSingleURLFromMockTweet(text string, urlText string) *Resource {
	config := MakeConfiguration()
	config.DownloadLinkAttachments = true
	// false for followHTMLRedirect because we need to test the features in the suite; in production it would be true
	config.FollowHTMLRedirects = false
	hr := HarvestResourceWithConfig(urlText, config)
	suite.NotNil(hr, "The harvested resources should not be Nil")
	return hr
}

func (suite *LinkSuite) TestInvalidlyFormattedURLs() {
	hr := suite.harvestSingleURLFromMockTweet("Test an invalidly formatted URL %s in a mock tweet", "https://t")
	isURLValid, isDestValid := hr.IsValid()
	suite.False(isURLValid, "URL should have invalid format")
	suite.False(isDestValid, "URL should have invalid destination")
	suite.Nil(hr.InspectionResults(), "No content should be available")
}

func (suite *LinkSuite) TestInvalidDestinationURLs() {
	hr := suite.harvestSingleURLFromMockTweet("Test a validly formatted URL %s but with invalid destination in a mock tweet", "https://t.co/fDxPF")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.False(isDestValid, "URL should have invalid destination")
	suite.Equal(hr.httpStatusCode, 404)
	suite.Nil(hr.InspectionResults(), "No content should be available")
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
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	suite.NotNil(hr.InspectionResults(), "Inspection results should be available")

	content := hr.InspectionResults()
	value, _ := content.GetOpenGraphMetaTag("site_name")
	suite.Equal(value, "Netspective")

	value, _ = content.GetOpenGraphMetaTag("title")
	suite.Equal(value, "Safety, privacy, and security focused technology consulting")

	value, _ = content.GetOpenGraphMetaTag("description")
	suite.Equal(value, "Software, technology, and management consulting focused on firms im pacted by FDA, ONC, NIST or other safety, privacy, and security regulations")
}

func (suite *LinkSuite) TestIgnoreRules() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore", "https://t.co/xNzrxkHE1u")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	isIgnored, ignoreReason := hr.IsIgnored()
	suite.True(isIgnored, "URL should be ignored (skipped)")
	suite.Equal(ignoreReason, "Matched Ignore Rule `^https://twitter.com/(.*?)/status/(.*)$`")
	suite.Nil(hr.InspectionResults(), "No content should be available")
}

func (suite *LinkSuite) TestResolvedURLRedirectedThroughHTMLProperly() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to resolve via <meta http-equiv='refresh' content='delay;url='>, with utm_* params", "http://bit.ly/lectio_harvester_resource_test03")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	isIgnored, _ := hr.IsIgnored()
	suite.False(isIgnored, "URL should not be ignored")
	isHTMLRedirect, htmlRedirectURLText := hr.IsHTMLRedirect()
	suite.True(isHTMLRedirect, "There should have been an HTML redirect requested through <meta http-equiv='refresh' content='delay;url='>")
	suite.Equal(htmlRedirectURLText, "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.NotNil(hr.InspectionResults(), "Inspection results should be available")

	// at this point we want to get the "new" (redirected) and test it
	config := MakeConfiguration()
	config.FollowHTMLRedirects = true
	config.DownloadLinkAttachments = true
	redirectedHR := HarvestResourceWithConfig("http://bit.ly/lectio_harvester_resource_test03", config)
	suite.NotNil(redirectedHR.ReferredByResource(), hr, "The referral resource should be the same as the original")
	isURLValid, isDestValid = redirectedHR.IsValid()
	suite.True(isURLValid, "Redirected URL should be formatted validly")
	suite.True(isDestValid, "Redirected URL should have valid destination")
	isIgnored, _ = redirectedHR.IsIgnored()
	suite.False(isIgnored, "Redirected URL should not be ignored")
	isCleaned, _ := redirectedHR.IsCleaned()
	suite.True(isCleaned, "Redirected URL should be 'cleaned'")
	finalURL, resolvedURL, cleanedURL := redirectedHR.GetURLs()
	suite.Equal(resolvedURL.String(), "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(cleanedURL.String(), "https://www.netspective.com/")
	suite.Equal(finalURL.String(), cleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(redirectedHR.InspectionResults(), "Inspection results should be available")
}

func (suite *LinkSuite) TestResolvedURLCleaned() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test01")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	isIgnored, _ := hr.IsIgnored()
	suite.False(isIgnored, "URL should not be ignored")
	isCleaned, _ := hr.IsCleaned()
	suite.True(isCleaned, "URL should be 'cleaned'")
	finalURL, resolvedURL, cleanedURL := hr.GetURLs()
	suite.Equal(resolvedURL.String(), "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(cleanedURL.String(), "https://www.netspective.com/")
	suite.Equal(finalURL.String(), cleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(hr.InspectionResults(), "Inspection results should be available")
}

func (suite *LinkSuite) TestResolvedURLCleanedKeys() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test02")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	isIgnored, _ := hr.IsIgnored()
	suite.False(isIgnored, "URL should not be ignored")
	isCleaned, _ := hr.IsCleaned()
	suite.True(isCleaned, "URL should be 'cleaned'")
	finalURL, resolvedURL, cleanedURL := hr.GetURLs()
	suite.Equal(resolvedURL.String(), "https://www.netspective.com/solutions/opsfolio/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(cleanedURL.String(), "https://www.netspective.com/solutions/opsfolio/")
	suite.Equal(finalURL.String(), cleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(hr.InspectionResults(), "Inspection results should be available")
	suite.Equal(hr.GloballyUniqueKey(), "c3ac941bc19188497805cbe583ff8d122ac663d6")
}

/*
func (suite *LinkSuite) TestResolvedURLCleanedSerializer() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore, with utm_* params", "http://bit.ly/lectio_harvester_resource_test02")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	isIgnored, _ := hr.IsIgnored()
	suite.False(isIgnored, "URL should not be ignored")
	isCleaned, _ := hr.IsCleaned()
	suite.True(isCleaned, "URL should be 'cleaned'")
	finalURL, resolvedURL, cleanedURL := hr.GetURLs()
	suite.Equal(resolvedURL.String(), "https://www.netspective.com/solutions/opsfolio/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.LinkSuite")
	suite.Equal(cleanedURL.String(), "https://www.netspective.com/solutions/opsfolio/")
	suite.Equal(finalURL.String(), cleanedURL.String(), "finalURL should be same as cleanedURL")

	suite.NotNil(hr.InspectionResults(), "Inspection results should be available")

	err := suite.harvested.Serialize(suite.serializer)
	suite.NoError(err, "Serialization should have occurred without error")

	_, found := suite.markdown[finalURL.String()]
	suite.True(found, "Markdown should have been serialized")
}
*/
func (suite *LinkSuite) TestResolvedURLNotCleaned() {
	hr := suite.harvestSingleURLFromMockTweet("Test a good URL %s which will redirect to a URL we want to ignore", "https://t.co/ELrZmo81wI")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	isIgnored, _ := hr.IsIgnored()
	suite.False(isIgnored, "URL should not be ignored")
	isCleaned, _ := hr.IsCleaned()
	suite.False(isCleaned, "URL should not have been 'cleaned'")
	finalURL, resolvedURL, cleanedURL := hr.GetURLs()
	suite.Equal(resolvedURL.String(), "https://www.foxnews.com/lifestyle/photo-of-donald-trump-look-alike-in-spain-goes-viral")
	suite.Equal(finalURL.String(), resolvedURL.String(), "finalURL should be same as resolvedURL")
	suite.Nil(cleanedURL, "cleanedURL should be empty")

	content := hr.InspectionResults()
	suite.NotNil(content, "The destination content should be available")
	suite.True(content.IsValid(), "The destination content should be valid")
	suite.True(content.IsHTML(), "The destination content should be HTML")
	suite.False(content.WasDownloaded(), "Because the destination was HTML, it should not have required to be downloaded")
}

func (suite *LinkSuite) TestResolvedDocumentURLNotCleaned() {
	hr := suite.harvestSingleURLFromMockTweet("Check out the PROV-O specification document %s, which should resolve to an 'attachment' style URL", "http://ceur-ws.org/Vol-1401/paper-05.pdf")
	isURLValid, isDestValid := hr.IsValid()
	suite.True(isURLValid, "URL should be formatted validly")
	suite.True(isDestValid, "URL should have valid destination")
	isIgnored, _ := hr.IsIgnored()
	suite.False(isIgnored, "URL should not be ignored")
	isCleaned, _ := hr.IsCleaned()
	suite.False(isCleaned, "URL should not have been 'cleaned'")
	finalURL, resolvedURL, cleanedURL := hr.GetURLs()
	suite.Equal(resolvedURL.String(), "http://ceur-ws.org/Vol-1401/paper-05.pdf")
	suite.Equal(finalURL.String(), resolvedURL.String(), "finalURL should be same as resolvedURL")
	suite.Nil(cleanedURL, "cleanedURL should be empty")

	content := hr.InspectionResults()
	suite.NotNil(content, "The destination content should be available")
	suite.True(content.IsValid(), "The destination content should be valid")
	suite.True(content.WasDownloaded(), "Because the destination wasn't HTML, it should have been downloaded")
	suite.Equal(content.attachment.fileType.Extension, "pdf")

	fileExists := false
	if _, err := os.Stat(content.attachment.destPath); err == nil {
		fileExists = true
	}
	suite.True(fileExists, "File %s should exist", content.attachment.destPath)
	suite.Equal(path.Ext(content.attachment.destPath), ".pdf", "File's extension should be .pdf")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(LinkSuite))
}
