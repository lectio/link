package link

type IssueCode string

const (
	MatchesIgnorePolicy       IssueCode = "LINKW-0100"
	URLStructureInvalid       IssueCode = "LINKE-0200"
	InvalidHTTPRespStatusCode IssueCode = "LINKE-0201"
	FinalURLNilOrEmpty        IssueCode = "LINKE-0300"
)

// Issue is a structured problem identification with context information
type Issue interface {
	IssueContext() interface{} // this will be the Link object plus location (item index, etc.), it's kept generic so it doesn't require package dependency
	IssueCode() IssueCode      // useful to uniquely identify a particular code
	Issue() string             // the

	IsError() bool   // this issue is an error
	IsWarning() bool // this issue is a warning
}

// Issues packages multiple issues into a container
type Issues interface {
	Issues() []Issue
	IssueCounts() (uint, uint, uint)
	HandleIssues(errorHandler func(Issue), warningHandler func(Issue))
}

type issue struct {
	context  *HarvestedLink
	code     IssueCode
	message  string
	isError  bool
	children []Issue
}

func newIssue(link *HarvestedLink, code IssueCode, message string, isError bool) Issue {
	result := new(issue)
	result.context = link
	result.code = code
	result.message = message
	result.isError = isError
	return result
}

func (i issue) IssueContext() interface{} {
	return i.context
}

func (i issue) IssueCode() IssueCode {
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
