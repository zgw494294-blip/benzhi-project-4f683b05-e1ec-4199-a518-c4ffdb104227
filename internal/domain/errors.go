package domain

import "fmt"

type ErrorCode string

const (
	CodeValidation ErrorCode = "VALIDATION"
	CodeConflict   ErrorCode = "CONFLICT"
	CodeNotFound   ErrorCode = "NOT_FOUND"
	CodeForbidden  ErrorCode = "FORBIDDEN"
	CodeCorrupt    ErrorCode = "CORRUPT"
)

type BusinessError struct {
	Code    ErrorCode
	Message string
}

func (e *BusinessError) Error() string { return e.Message }

func Invalid(format string, args ...any) error {
	return &BusinessError{Code: CodeValidation, Message: fmt.Sprintf(format, args...)}
}

func Conflict(format string, args ...any) error {
	return &BusinessError{Code: CodeConflict, Message: fmt.Sprintf(format, args...)}
}

func Forbidden(format string, args ...any) error {
	return &BusinessError{Code: CodeForbidden, Message: fmt.Sprintf(format, args...)}
}

func NotFound(format string, args ...any) error {
	return &BusinessError{Code: CodeNotFound, Message: fmt.Sprintf(format, args...)}
}
