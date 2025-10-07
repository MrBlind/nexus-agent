package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/mrblind/nexus-agent/pkg/trace"
)

const requestIDHeader = "X-Request-ID"

// RequestID ensures every request has a request ID in context and response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = trace.RandomID()
		}
		ctx := trace.WithRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Set(string(trace.RequestIDKey), id)
		c.Next()
	}
}
