package errors

import (
	"fmt"
	"runtime"
)

type AppError struct {
	Code    int
	Message string
	Stack   []string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Stack:   captureStack(),
	}
}

func Wrap(code int, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Stack:   captureStack(),
	}
}

func captureStack() []string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	stack := make([]string, 0, n)
	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return stack
}

const (
	CodeSuccess         = 0
	CodeBadRequest      = 1000
	CodeUnauthorized    = 1001
	CodeForbidden       = 1002
	CodeNotFound        = 1003
	CodeConflict        = 1004
	CodeTooManyRequests = 1005
	CodeInternal        = 2000
	CodeDatabaseError   = 2001
	CodeDependencyError = 2002
)

func BadRequest(msg string) *AppError      { return New(CodeBadRequest, msg) }
func Unauthorized(msg string) *AppError    { return New(CodeUnauthorized, msg) }
func Forbidden(msg string) *AppError       { return New(CodeForbidden, msg) }
func NotFound(msg string) *AppError        { return New(CodeNotFound, msg) }
func Conflict(msg string) *AppError        { return New(CodeConflict, msg) }
func TooManyRequests(msg string) *AppError { return New(CodeTooManyRequests, msg) }
func Internal(msg string) *AppError        { return New(CodeInternal, msg) }
func DatabaseError(err error) *AppError    { return Wrap(CodeDatabaseError, "database error", err) }
