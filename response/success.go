package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Meta contains response metadata.
type Meta struct {
	TraceID   string `json:"traceId,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

func getTraceID(c *gin.Context) string {
	if traceID := c.GetString("traceId"); traceID != "" {
		return traceID
	}
	return ""
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
			TraceID:   getTraceID(c),
			RequestID: c.GetHeader("X-Request-ID"),
		},
	})
}

// Created sends a 201 Created JSON response.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Data: data,
		Meta: Meta{
			TraceID:   getTraceID(c),
			RequestID: c.GetHeader("X-Request-ID"),
		},
	})
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
