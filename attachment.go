package link

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	filetype "github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
)

// Attachment manages any content that was downloaded for further inspection
type Attachment struct {
	URL           *url.URL   `json:"url"`
	DestPath      string     `json:"destPath"`
	FileType      types.Type `json:"fileType"`
	DownloadError error      `json:"downloadError,omitempty"`
	FileTypeError error      `json:"fileTypeError,omitempty"`
}

// IsValid returns true if there are no errors
func (a Attachment) IsValid() bool {
	if a.DownloadError != nil {
		return false
	}
	if a.FileTypeError != nil {
		return false
	}

	return true
}

// Delete removes the file that was downloaded
func (a *Attachment) Delete() {
	os.Remove(a.DestPath)
}

// download will download the URL as an "attachment" to a local file.
// It's efficient because it will write as it downloads and not load the whole file into memory.
func downloadFile(url *url.URL, resp *http.Response, destFile *os.File) *Attachment {
	result := new(Attachment)
	result.URL = url

	defer destFile.Close()
	defer resp.Body.Close()
	result.DestPath = destFile.Name()
	_, err := io.Copy(destFile, resp.Body)
	if err != nil {
		result.DownloadError = err
		return result
	}
	destFile.Close()

	// Open the just-downloaded file again since it was closed already
	file, err := os.Open(result.DestPath)
	if err != nil {
		result.FileTypeError = err
		return result
	}

	// We only have to pass the file header = first 261 bytes
	head := make([]byte, 261)
	file.Read(head)
	file.Close()

	result.FileType, result.FileTypeError = filetype.Match(head)
	if result.FileTypeError == nil {
		// change the extension so that it matches the file type we found
		currentPath := result.DestPath
		currentExtension := path.Ext(currentPath)
		newPath := currentPath[0:len(currentPath)-len(currentExtension)] + "." + result.FileType.Extension
		os.Rename(currentPath, newPath)
		result.DestPath = newPath
	}

	return result
}

// downloadTemp will download the URL as an "attachment" to a temporary file.
func downloadTemp(url *url.URL, resp *http.Response, tempPattern string) *Attachment {
	destFile, err := ioutil.TempFile(os.TempDir(), tempPattern)

	if err != nil {
		result := new(Attachment)
		result.URL = url
		result.DownloadError = err
		return result
	}

	return downloadFile(url, resp, destFile)
}

// download will download the URL as an "attachment" to named file.
func download(url *url.URL, resp *http.Response, pathAndFileName string) *Attachment {
	destFile, err := os.Create(pathAndFileName)

	if err != nil {
		result := new(Attachment)
		result.URL = url
		result.DownloadError = err
		return result
	}

	return downloadFile(url, resp, destFile)
}
