package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorDetail contains field-level validation error details.
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// ErrorBody contains the specific error details.
type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorResponse standardizes the JSON response payload structure for errors.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// CustomError sends a custom error response.
func CustomError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

// BadRequest sends a 400 Bad Request JSON response.
func BadRequest(c *gin.Context, code, message string) {
	CustomError(c, http.StatusBadRequest, code, message)
}

// Unauthorized sends a 401 Unauthorized JSON response.
func Unauthorized(c *gin.Context, code, message string) {
	CustomError(c, http.StatusUnauthorized, code, message)
}

// Forbidden sends a 403 Forbidden JSON response.
func Forbidden(c *gin.Context, code, message string) {
	CustomError(c, http.StatusForbidden, code, message)
}

// NotFound sends a 404 Not Found JSON response.
func NotFound(c *gin.Context, code, message string) {
	CustomError(c, http.StatusNotFound, code, message)
}

// InternalServerError sends a 500 Internal Server Error JSON response.
func InternalServerError(c *gin.Context, code, message string) {
	CustomError(c, http.StatusInternalServerError, code, message)
}

// CustomErrorWithDetails sends a custom error response with details array.
func CustomErrorWithDetails(c *gin.Context, status int, code, message string, details []ErrorDetail) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
	default:
		return fe.Error()
	}
}

// ValidationError formats and sends a 400 Bad Request for validation errors.
// If err is not a validator.ValidationErrors, it falls back to standard BadRequest.
func ValidationError(c *gin.Context, err error) {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		details := make([]ErrorDetail, 0, len(validationErrors))
		for _, e := range validationErrors {
			details = append(details, ErrorDetail{
				Field:   e.Field(),
				Code:    e.Tag(),
				Message: validationMessage(e),
			})
		}
		CustomErrorWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", details)
		return
	}
	BadRequest(c, "BAD_REQUEST", err.Error())
}
