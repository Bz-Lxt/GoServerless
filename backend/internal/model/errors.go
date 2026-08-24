package model

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrInvalid      = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrTimeout      = errors.New("execution timeout")
	ErrUnavailable  = errors.New("runtime unavailable")
	ErrBuildFailed  = errors.New("build failed")
	ErrNotReady     = errors.New("function not ready")
	ErrTooLarge     = errors.New("payload too large")
	ErrConcurrency  = errors.New("concurrency limit exceeded")
)

type Error struct {
	Code    string
	Message string
	Status  int
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

func NewError(code, msg string, status int, cause error) *Error {
	return &Error{Code: code, Message: msg, Status: status, Cause: cause}
}

func Invalid(msg string) *Error {
	return NewError("INVALID_INPUT", msg, http.StatusBadRequest, ErrInvalid)
}

func NotFound(resource, name string) *Error {
	return NewError("NOT_FOUND", fmt.Sprintf("%s %q not found", resource, name), http.StatusNotFound, ErrNotFound)
}

func Conflict(msg string) *Error {
	return NewError("CONFLICT", msg, http.StatusConflict, ErrConflict)
}

func Unauthorized(msg string) *Error {
	return NewError("UNAUTHORIZED", msg, http.StatusUnauthorized, ErrUnauthorized)
}

func Timeout(msg string) *Error {
	return NewError("TIMEOUT", msg, http.StatusGatewayTimeout, ErrTimeout)
}

func Unavailable(msg string) *Error {
	return NewError("UNAVAILABLE", msg, http.StatusServiceUnavailable, ErrUnavailable)
}

func BuildFailed(msg string) *Error {
	return NewError("BUILD_FAILED", msg, http.StatusUnprocessableEntity, ErrBuildFailed)
}

func NotReady(name string) *Error {
	return NewError("NOT_READY", fmt.Sprintf("function %q is not READY", name), http.StatusConflict, ErrNotReady)
}

func TooLarge(msg string) *Error {
	return NewError("TOO_LARGE", msg, http.StatusRequestEntityTooLarge, ErrTooLarge)
}

func Concurrency(name string) *Error {
	return NewError("CONCURRENCY", fmt.Sprintf("function %q exceeded concurrency limit", name), http.StatusTooManyRequests, ErrConcurrency)
}

func HTTPStatus(err error) int {
	var e *Error
	if errors.As(err, &e) && e.Status > 0 {
		return e.Status
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrBuildFailed):
		return http.StatusBadRequest
	case errors.Is(err, ErrConflict), errors.Is(err, ErrNotReady):
		return http.StatusConflict
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, ErrTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrConcurrency):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) && e.Code != "" {
		return e.Code
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, ErrInvalid):
		return "INVALID_INPUT"
	case errors.Is(err, ErrConflict):
		return "CONFLICT"
	case errors.Is(err, ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, ErrTimeout):
		return "TIMEOUT"
	default:
		return "INTERNAL"
	}
}

func MessageOf(err error) string {
	var e *Error
	if errors.As(err, &e) && e.Message != "" {
		return e.Message
	}
	if err == nil {
		return ""
	}
	return err.Error()
}
