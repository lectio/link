package link

type ErrorCode string

const (
	MatchesIgnorePolicyErrorCode   ErrorCode = "LINK-0100"
	URLStructureInvalidErrorCode   ErrorCode = "LINK-0200"
	URLDestinationInvalidErrorCode ErrorCode = "LINK-0201"
	FinalURLNilOrEmptyErrorCode    ErrorCode = "LINK-0300"
)

// Error is a structured problem identification with context information
type Error interface {
	ErrorContext() interface{} // this will be the Link object plus location (item index, etc.), it's kept generic so it doesn't require package dependency
	ErrorCode() ErrorCode      // useful to uniquely identify a particular code
	Error() string             // statifies the Go error standard interface
}

type basicError struct {
	context *HarvestedLink
	code    ErrorCode
	message string
}

func newError(link *HarvestedLink, code ErrorCode, message string) Error {
	result := new(basicError)
	result.context = link
	result.code = code
	result.message = message
	return result
}

func (e basicError) ErrorContext() interface{} {
	return e.context
}

func (e basicError) ErrorCode() ErrorCode {
	return e.code
}

func (e basicError) Error() string {
	return e.message
}
