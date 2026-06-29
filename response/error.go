package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody contains the specific error details.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse standardizes the JSON response payload structure for errors.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

// CustomError sends a custom error response.
func CustomError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
		Meta: Meta{
			TraceID:   getTraceID(c),
			RequestID: c.GetHeader("X-Request-ID"),
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
