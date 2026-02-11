package errors

import "fmt"

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Wrap(err error) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		StatusCode: e.StatusCode,
		Err:        err,
	}
}

func (e *AppError) Unwrap() error {
	return e.Err
}

var (
	ErrNotFound            = &AppError{Code: "NOT_FOUND", Message: "Resource not found", StatusCode: 404}
	ErrUnauthorized        = &AppError{Code: "UNAUTHORIZED", Message: "Unauthorized", StatusCode: 401}
	ErrForbidden           = &AppError{Code: "FORBIDDEN", Message: "Forbidden", StatusCode: 403}
	ErrTokenExpired        = &AppError{Code: "TOKEN_EXPIRED", Message: "Token has expired", StatusCode: 401}
	ErrInvalidToken        = &AppError{Code: "INVALID_TOKEN", Message: "Invalid token", StatusCode: 401}
	ErrRateLimitExceeded   = &AppError{Code: "RATE_LIMIT", Message: "Rate limit exceeded", StatusCode: 429}
	ErrToolExecutionFailed = &AppError{Code: "TOOL_EXEC_FAILED", Message: "Tool execution failed", StatusCode: 500}
	ErrToolNotAllowed      = &AppError{Code: "TOOL_NOT_ALLOWED", Message: "Tool not allowed for this token", StatusCode: 403}
	ErrPendingApproval     = &AppError{Code: "PENDING_APPROVAL", Message: "Action pending approval", StatusCode: 202}
	ErrValidation          = &AppError{Code: "VALIDATION_ERROR", Message: "Validation failed", StatusCode: 400}
	ErrInternal            = &AppError{Code: "INTERNAL_ERROR", Message: "Internal server error", StatusCode: 500}
)
