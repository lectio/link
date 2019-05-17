package link

import (
	"fmt"
	"golang.org/x/xerrors"
)

// InvalidHTTPRespStatusCodeError is used as Error.Code when the traversed URL returns a non-200 status code
type InvalidHTTPRespStatusCodeError struct {
	Message        string
	Code           int
	HTTPStatusCode int
	frame          xerrors.Frame
}

// FormatError will print a simple message to the Printer object. This will be what you see when you Println or use %s/%v in a formatted print statement.
func (e InvalidHTTPRespStatusCodeError) FormatError(p xerrors.Printer) error {
	p.Printf("LECTIOLINK-%d %s", e.Code, e.Message)
	e.frame.Format(p)
	return nil
}

// Format provide backwards compatibility with pre-xerrors package
func (e InvalidHTTPRespStatusCodeError) Format(f fmt.State, c rune) {
	xerrors.FormatError(e, f, c)
}

// Format provide backwards compatibility with pre-xerrors package
func (e InvalidHTTPRespStatusCodeError) Error() string {
	return fmt.Sprint(e)
}

// URLStructureInvalidError is used as Error.Code when the URL cannot be parsed
type URLStructureInvalidError struct {
	Message string
	Code    int
	frame   xerrors.Frame
}

// FormatError will print a simple message to the Printer object. This will be what you see when you Println or use %s/%v in a formatted print statement.
func (e URLStructureInvalidError) FormatError(p xerrors.Printer) error {
	p.Printf("LECTIOLINK-%d %s", e.Code, e.Message)
	e.frame.Format(p)
	return nil
}

// Format provide backwards compatibility with pre-xerrors package
func (e URLStructureInvalidError) Format(f fmt.State, c rune) {
	xerrors.FormatError(e, f, c)
}

// Format provide backwards compatibility with pre-xerrors package
func (e URLStructureInvalidError) Error() string {
	return fmt.Sprint(e)
}
