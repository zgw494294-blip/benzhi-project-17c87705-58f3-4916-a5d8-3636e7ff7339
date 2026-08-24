package domain

import "fmt"

type ErrorCode string

const (
	CodeValidation      ErrorCode = "VALIDATION_ERROR"
	CodeConflict        ErrorCode = "VERSION_CONFLICT"
	CodeInvalidState    ErrorCode = "INVALID_STATE"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeForbidden       ErrorCode = "FORBIDDEN"
	CodeGateFailed      ErrorCode = "GATE_FAILED"
	CodeIntegrityFailed ErrorCode = "INTEGRITY_FAILED"
)

type RuleError struct {
	Code    ErrorCode
	Field   string
	Message string
}

func (e *RuleError) Error() string { return e.Message }
func NewRuleError(code ErrorCode, field, format string, args ...any) error {
	return &RuleError{Code: code, Field: field, Message: fmt.Sprintf(format, args...)}
}
