package link

import "fmt"

const (
	MatchesIgnorePolicy       string = "LINKW-0100"
	URLStructureInvalid       string = "LINKE-0200"
	InvalidHTTPRespStatusCode string = "LINKE-0201"
	FinalURLNilOrEmpty        string = "LINKE-0300"
)

// Issue is a structured problem identification with context information
type Issue interface {
	IssueContext() interface{} // this will be the Link object plus location (item index, etc.), it's kept generic so it doesn't require package dependency
	IssueCode() string         // useful to uniquely identify a particular code
	Issue() string             // the

	IsError() bool   // this issue is an error
	IsWarning() bool // this issue is a warning
}

// Issues packages multiple issues into a container
type Issues interface {
	ErrorsAndWarnings() []Issue
	IssueCounts() (uint, uint, uint)
	HandleIssues(errorHandler func(Issue), warningHandler func(Issue))
}

type issue struct {
	context string
	code    string
	message string
	isError bool
}

func newIssue(context string, code string, message string, isError bool) Issue {
	result := new(issue)
	result.context = context
	result.code = code
	result.message = message
	result.isError = isError
	return result
}

func newHTTPResponseIssue(context string, httpRespStatusCode int, message string, isError bool) Issue {
	result := new(issue)
	result.context = context
	result.code = fmt.Sprintf("%s-HTTP-%d", InvalidHTTPRespStatusCode, httpRespStatusCode)
	result.message = message
	result.isError = isError
	return result
}

func (i issue) IssueContext() interface{} {
	return i.context
}

func (i issue) IssueCode() string {
	return i.code
}

func (i issue) Issue() string {
	return i.message
}

func (i issue) IsError() bool {
	return i.isError
}

func (i issue) IsWarning() bool {
	return !i.isError
}

// Error satisfies the Go error contract
func (i issue) Error() string {
	return i.message
}
