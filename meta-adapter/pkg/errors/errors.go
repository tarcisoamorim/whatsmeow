package errors

import (
	"fmt"
	"net/http"

	"github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/models"
)

// Common error codes following Meta API conventions
const (
	ErrCodeInvalidParameter    = 100
	ErrCodeAccessDenied        = 200
	ErrCodePermissionDenied    = 220
	ErrCodeRateLimitExceeded   = 130429
	ErrCodeMessageUndelivered  = 131026
	ErrCodeAccountNotConnected = 131051
	ErrCodeMediaDownloadError  = 131052
	ErrCodeMediaUploadError    = 131053
	ErrCodeInvalidPhoneNumber  = 131031
	ErrCodeRecipientNotAvail   = 131056
	ErrCodeServiceUnavailable  = 2
	ErrCodeInternalError       = 1
)

// AdapterError represents an error with Meta API compatible format
type AdapterError struct {
	HTTPStatus   int
	ErrorCode    int
	ErrorMessage string
	ErrorType    string
	ErrorSubcode int
}

func (e *AdapterError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.ErrorCode, e.ErrorType, e.ErrorMessage)
}

// ToMetaError converts AdapterError to Meta API error response
func (e *AdapterError) ToMetaError() models.ErrorResponse {
	return models.ErrorResponse{
		Error: models.ErrorDetail{
			Message:      e.ErrorMessage,
			Type:         e.ErrorType,
			Code:         e.ErrorCode,
			ErrorSubcode: e.ErrorSubcode,
		},
	}
}

// NewError creates a new AdapterError
func NewError(httpStatus, code int, errorType, message string) *AdapterError {
	return &AdapterError{
		HTTPStatus:   httpStatus,
		ErrorCode:    code,
		ErrorType:    errorType,
		ErrorMessage: message,
	}
}

// Common errors
var (
	ErrInvalidRequest = NewError(
		http.StatusBadRequest,
		ErrCodeInvalidParameter,
		"invalid_request",
		"The request is invalid or malformed",
	)

	ErrUnauthorized = NewError(
		http.StatusUnauthorized,
		ErrCodeAccessDenied,
		"authentication_error",
		"Invalid or missing API key",
	)

	ErrNotConnected = NewError(
		http.StatusServiceUnavailable,
		ErrCodeAccountNotConnected,
		"connection_error",
		"WhatsApp session is not connected. Please authenticate first.",
	)

	ErrRateLimitExceeded = NewError(
		http.StatusTooManyRequests,
		ErrCodeRateLimitExceeded,
		"rate_limit_exceeded",
		"Too many requests. Please slow down.",
	)

	ErrInternalServer = NewError(
		http.StatusInternalServerError,
		ErrCodeInternalError,
		"internal_error",
		"An internal error occurred",
	)

	ErrMediaDownload = NewError(
		http.StatusBadRequest,
		ErrCodeMediaDownloadError,
		"media_download_error",
		"Failed to download media",
	)

	ErrMediaUpload = NewError(
		http.StatusBadRequest,
		ErrCodeMediaUploadError,
		"media_upload_error",
		"Failed to upload media",
	)

	ErrInvalidPhoneNumber = NewError(
		http.StatusBadRequest,
		ErrCodeInvalidPhoneNumber,
		"invalid_phone_number",
		"The phone number is invalid",
	)

	ErrRecipientUnavailable = NewError(
		http.StatusBadRequest,
		ErrCodeRecipientNotAvail,
		"recipient_unavailable",
		"The recipient is not available on WhatsApp",
	)
)

// WrapError wraps an error with additional context
func WrapError(err error, message string) *AdapterError {
	if ae, ok := err.(*AdapterError); ok {
		ae.ErrorMessage = fmt.Sprintf("%s: %s", message, ae.ErrorMessage)
		return ae
	}
	return NewError(
		http.StatusInternalServerError,
		ErrCodeInternalError,
		"internal_error",
		fmt.Sprintf("%s: %v", message, err),
	)
}

// ValidationError creates a validation error
func ValidationError(field, message string) *AdapterError {
	return NewError(
		http.StatusBadRequest,
		ErrCodeInvalidParameter,
		"validation_error",
		fmt.Sprintf("Invalid %s: %s", field, message),
	)
}
