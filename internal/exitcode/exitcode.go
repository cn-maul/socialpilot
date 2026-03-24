package exitcode

import "fmt"

const (
	Success    = 0
	InvalidArg = 1
	NetworkErr = 2
	Database   = 3
	LLMParse   = 4
)

type CodedError struct {
	Code int
	Msg  string
	Err  error
}

func (e *CodedError) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	if e.Msg == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Msg, e.Err)
}

func (e *CodedError) Unwrap() error { return e.Err }
func (e *CodedError) ExitCode() int { return e.Code }

func New(code int, msg string, err error) error {
	return &CodedError{Code: code, Msg: msg, Err: err}
}
