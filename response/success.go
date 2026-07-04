package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Meta contains response metadata.
type Meta struct {
	RequestID string `json:"requestId,omitempty"`
}

func getRequestID(c *gin.Context) string {
	if requestID := c.GetString("requestID"); requestID != "" {
		return requestID
	}
	return c.GetHeader("X-Request-ID")
}

// SuccessBody contains the success response data.
type SuccessBody struct {
	Data interface{} `json:"data"`
}

// SuccessResponse standardizes the JSON response payload structure for success.
type SuccessResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

// OK sends a 200 OK JSON response.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Data: data,
		Meta: Meta{
			RequestID: getRequestID(c),
		},
	})
}

// Created sends a 201 Created JSON response.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Data: data,
		Meta: Meta{
			RequestID: getRequestID(c),
		},
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
