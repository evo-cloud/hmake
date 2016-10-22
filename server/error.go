package server

// Error is dedicated type of errors on server
type Error struct {
	Cause   error
	Message string
}

// NewError creates an error with a message
func NewError(msg string) *Error {
	return &Error{Message: msg}
}

// NewErrorCause creates an error with a cause
func NewErrorCause(err error) *Error {
	if err == nil {
		return nil
	}
	return &Error{Cause: err}
}

// NewErrorMsgCause creates an error with cause and message
func NewErrorMsgCause(msg string, err error) *Error {
	return &Error{Cause: err, Message: msg}
}

// Error implements error
func (e *Error) Error() string {
	if e.Message == "" && e.Cause != nil {
		return e.Cause.Error()
	}
	msg := e.Message
	if e.Cause != nil {
		if msg != "" {
			msg += " by "
		}
		msg += e.Cause.Error()
	}
	return msg
}
