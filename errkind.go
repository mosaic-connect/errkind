// Package errkind is used to create and detect specific kinds of errors,
// based on single method interfaces that the errors support.
//
// Supported interfaces
//
// Temporary errors are detected using the ``temporaryer'' interface. Some
// errors in the Go standard library implement this interface. (See net.AddrError,
// net.DNSConfigError, and net.DNSError for examples).
//  type temporaryer interface {
//      Temporary() bool
//  }
//
// Some packages return errors which implement the `coder`
// interface, which allows the error to report an application-specific
// error condition.
//  type coder interface {
//      Code() string
//  }
// The AWS SDK for Go is a popular third party library that follows this
// convention.
//
// In addition some third party packages (including the AWS SDK) follow the
// convention of reporting HTTP status values using the `statusCoder` interface.
//  type statusCoder interface {
//      StatusCode() int
//  }
//
// The publicMessager interface identifies an error as having a message suitable
// for displaying to a requesting client. The error message does not contain any
// implementation details that could leak sensitive information.
//  type publicMessager interface {
//      PublicMessage()
//  }
//
// The publicStatusCoder interface identifies an error has having a status code
// suitable for returning to a requesting client.
//  type publicStatusCoder interface {
//      PublicStatusCode()
//  }
//
package errkind

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-stack/stack"
	"github.com/jjeffery/errors"
)

// cause is an interface implemented by errors that have a cause error.
type causer interface {
	Cause() error
}

// temporaryer is an interface implemented by errors that communicate
// if they are temporary or not. Temporary errors can be retried.
type temporaryer interface {
	Temporary() bool
}

// coder is an interface implemented by errors that return a string code.
// Useful for checking AWS error codes.
type coder interface {
	Code() string
}

// statusCode is an interface implemented by errors that return an integer status code.
// Userful for checking AWS status codes.
type statusCoder interface {
	StatusCode() int
}

// publicMessager is an interface implemented by errors whose contents are suitable
// for returning to requesting clients. Their message does not include implementation details.
type publicMessager interface {
	PublicMessage()
}

// publicStatusCoder is an interface implemented by errors whose status code
// is public and can be returned to requesting clients.
type publicStatusCoder interface {
	PublicStatusCode()
}

// publicCoder is an interface implemented by errors whose error code is public
// and can be returned to requesting clients.
type publicCoder interface {
	PublicCode()
}

// HasCode determines whether the error has any of the codes associated with it.
func HasCode(err error, codes ...string) bool {
	err = errors.Cause(err)
	if err == nil {
		return false
	}
	if errCoder, ok := err.(coder); ok {
		errCode := errCoder.Code()
		for _, code := range codes {
			if errCode == code {
				return true
			}
		}
	}
	return false
}

// HasStatusCode determines whether the error has any of the statuses associated with it.
func HasStatusCode(err error, statusCodes ...int) bool {
	statusCode := StatusCode(err)
	for _, sc := range statusCodes {
		if statusCode == sc {
			return true
		}
	}
	return false
}

// StatusCode returns the status code associated with err, or
// zero if there is no status.
func StatusCode(err error) int {
	err = errors.Cause(err)
	if err == nil {
		return 0
	}
	if errStatusCoder, ok := err.(statusCoder); ok {
		return errStatusCoder.StatusCode()
	}
	return 0
}

// Status does the same thing as StatusCode.
//
// Deprecated: use StatusCode instead.
func Status(err error) int {
	return StatusCode(err)
}

// Code returns the string error code associated with err, or
// a blank string if there is no code.
func Code(err error) string {
	err = errors.Cause(err)
	if err == nil {
		return ""
	}
	if errCoder, ok := err.(coder); ok {
		return errCoder.Code()
	}
	return ""
}

// IsTemporary returns true for errors that indicate
// an error condition that may succeed if retried.
//
// An error is considered temporary if it implements
// the following interface and its Temporary method returns true.
//  type temporaryer interface {
//      Temporary() bool
//  }
func IsTemporary(err error) bool {
	err = errors.Cause(err)
	for err == nil {
		return false
	}
	if temporary, ok := err.(temporaryer); ok {
		return temporary.Temporary()
	}
	return false
}

// statusError implements error, statusCoder and publicer interfaces.
type statusError struct {
	message string
	status  int
}

func (s statusError) Error() string {
	return s.message
}

func (s statusError) StatusCode() int {
	return s.status
}

func (s statusError) PublicStatusCode() {}

func (s statusError) With(keyvals ...interface{}) errors.Error {
	return errors.Wrap(s).With(keyvals...)
}

// publicStatusError implements error, statusCoder and publicMessager interfaces.
type publicStatusError struct {
	statusError
}

func (s publicStatusError) PublicMessage() {}

// publicStatusCodeError implements error, statusCoder, coder and publicMessager interfaces.
type publicStatusCodeError struct {
	message string
	status  int
	code    string
}

func (s publicStatusCodeError) Error() string {
	if strings.ContainsAny(s.code, "\n\r\t \"'") {
		return fmt.Sprintf("%s code=%q", s.message, s.code)
	}
	return fmt.Sprintf("%s code=%s", s.message, s.code)
}

func (s publicStatusCodeError) Message() string {
	return s.message
}

func (s publicStatusCodeError) StatusCode() int {
	return s.status
}

func (s publicStatusCodeError) Code() string {
	return s.code
}

func (s publicStatusCodeError) PublicMessage() {}

func (s publicStatusCodeError) PublicStatusCode() {}

func (s publicStatusCodeError) PublicCode() {}

func (s publicStatusCodeError) With(keyvals ...interface{}) errors.Error {
	return errors.Wrap(s).With(keyvals...)
}

// makeMessage returns a string message based on a default message,
// and zero or more strings in the msg slice. If there is one or more
// non-blank messages in the msg slice, then they are concatenated and
// returned. Usually there will be one non-blank message. If there are no
// non-blank messages, then the default message is returned.
func makeMessage(defaultMsg string, msgs []string) string {
	var messages []string
	if len(msgs) > 0 {
		messages = make([]string, 0, len(msgs))
	}
	for _, msg := range msgs {
		msg = strings.TrimSpace(msg)
		if msg != "" {
			messages = append(messages, msg)
		}
	}
	if len(messages) == 0 {
		return defaultMsg
	}
	return strings.Join(messages, " ")
}

// Public returns an error with the message and status.
// The message should not contain any implementation details as
// it may be displayed to a requesting client.
//
// Note that if you attach any key/value pairs to the public
// error using the With method, then that will return a new error that
// is not public, as implementation details may be present in the key/value pairs.
// The cause of the new error, however, will still be public.
func Public(message string, status int) errors.Error {
	return publicStatusError{
		statusError{
			message: message,
			status:  status,
		},
	}
}

// PublicWithCode returns an error with the message, status and code.
// The code can be useful for indicating specific error conditions to
// a requesting client.
//
// The message and code should not contain any implementation details as
// it may be displayed to a requesting client.
//
// Note that if you attach any key/value pairs to the public
// error using the With method, then that will return a new error that
// is not public, as implementation details may be present in the key/value pairs.
// The cause of the new error, however, will still be public.
func PublicWithCode(message string, status int, code string) errors.Error {
	code = strings.TrimSpace(code)
	if code == "" {
		// no code supplied
		return Public(message, status)
	}
	return publicStatusCodeError{
		message: message,
		status:  status,
		code:    code,
	}
}

// HasPublicMessage returns true for errors that indicate
// that their message does not contain sensitive information
// and can be displayed to external clients.
//
// An error has a public message if it implements
// the following interface.
//  type publicMessager interface {
//      PublicMessage()
//  }
//
// It usually makes sense to obtain the cause of an error first
// before testing to see if it is public. Any public error that
// is wrapped using errors.Wrap, or errors.With will return a
// new error that is no longer public.
//  // get the cause of the error
//  err = errors.Cause(err)
//  if errkind.HasPublicMessage(err) {
//      // ... can provide err.Error() to the client
//  }
func HasPublicMessage(err error) bool {
	_, ok := err.(publicMessager)
	return ok
}

/*********************

TODO(jpj): maybe include in the public api

// HasPublicStatusCode returns true for errors that indicate
// that their status code does not contain sensitive information
// and can be displayed to external clients.
//
// An error has a public status code if it implements
// the following interface.
//  type publicStatusCoder interface {
//      PublicStatusCode()
//  }
func HasPublicStatusCode(err error) bool {
	_, ok := err.(publicStatusCoder)
	return ok
}

***************************/

// BadRequest returns an client error that has a status of 400 (bad request).
//
// The returned error has a PublicStatusCode() method, which indicates that the
// status code is public and can be returned to a client.
func BadRequest(msg ...string) errors.Error {
	return statusError{
		message: makeMessage("bad request", msg),
		status:  http.StatusBadRequest,
	}
}

// Unauthorized returns a client error that has a status of 401 (unauthorized).
//
// The returned error has a PublicStatusCode() method, which indicates that the
// status code is public and can be returned to a client.
func Unauthorized(msg ...string) errors.Error {
	return statusError{
		message: makeMessage("unauthorized", msg),
		status:  http.StatusUnauthorized,
	}
}

// Forbidden returns an error that has a status of 403 (forbidden).
//
// The returned error has a PublicStatusCode() method, which indicates that the
// status code is public and can be returned to a client.
func Forbidden(msg ...string) errors.Error {
	return statusError{
		message: makeMessage("forbidden", msg),
		status:  http.StatusForbidden,
	}
}

// NotFound returns an error that has a status of 404 (not found).
//
// The returned error has a PublicStatusCode() method, which indicates that the
// status code is public and can be returned to a client.
func NotFound(msg ...string) errors.Error {
	return statusError{
		message: makeMessage("not found", msg),
		status:  http.StatusNotFound,
	}
}

// NotImplemented returns an error with a status of 501 (not implemented).
//
// The returned error has a PublicStatusCode() method, which indicates that the
// status code is public and can be returned to a client.
func NotImplemented(msg ...string) errors.Error {
	return statusError{
		message: makeMessage("not implemented", msg),
		status:  http.StatusNotImplemented,
	}.With("caller", stack.Caller(1))
}

type temporaryError string

func (t temporaryError) Error() string {
	return string(t)
}

func (t temporaryError) Temporary() bool {
	return true
}

// Temporary returns an error that indicates it is temporary.
func Temporary(msg string) errors.Error {
	return errors.Wrap(temporaryError(msg))
}
