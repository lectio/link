package cache

import (
	"testing"

	"github.com/lectio/link"
	"github.com/stretchr/testify/suite"
)

type CacheSuite struct {
	suite.Suite
	config *link.Configuration
	cache  Cache
	keys   link.Keys
}

func (suite *CacheSuite) SetupSuite() {
	config := link.MakeConfiguration()
	keys := link.MakeDefaultKeys()
	cache, err := MakeFileCache("test", true, keys, config, config, config)
	if err != nil {
		panic(err)
	}
	suite.cache = cache
}

func (suite *CacheSuite) TearDownSuite() {
	suite.cache.Close()
}

func (suite *CacheSuite) TestInvalidlyFormattedURLs() {
	link, err := suite.cache.Get("https://t")
	suite.Nil(err, "There should be no cache error")
	suite.NotNil(link, "Link should not be nil")
	suite.False(link.IsURLValid, "URL should have invalid format")
	suite.False(link.IsDestValid, "URL should have invalid destination")
	suite.Nil(link.Content, "No content should be available")
}

func (suite *CacheSuite) TestResolvedURLCleaned() {
	link, err := suite.cache.Get("http://bit.ly/lectio_harvester_resource_test01")
	suite.Nil(err, "There should be no cache error")
	suite.True(link.IsURLValid, "URL should be formatted validly")
	suite.True(link.IsDestValid, "URL should have valid destination")
	suite.False(link.IsURLIgnored, "URL should not be ignored")
	suite.True(link.AreURLParamsCleaned, "URL should be 'cleaned'")
	suite.Equal(link.ResolvedURL.String(), "https://www.netspective.com/?utm_source=lectio_harvester_resource_test.go&utm_medium=go.TestSuite&utm_campaign=harvester.ResourceSuite")
	suite.Equal(link.CleanedURL.String(), "https://www.netspective.com/")
	suite.Equal(link.FinalizedURL.String(), link.CleanedURL.String(), "finalURL should be same as cleanedURL")
	suite.NotNil(link.Content, "Inspection results should be available")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(CacheSuite))
}
