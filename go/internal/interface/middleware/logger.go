package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrblind/nexus-agent/pkg/logger"
	"github.com/mrblind/nexus-agent/pkg/trace"
)

func Logger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		reqID, ok := c.Get(string(trace.RequestIDKey))
		id := ""
		if ok {
			s, _ := reqID.(string)
			id = s
		}
		entry := log.WithRequestID(id)

		entry.Info().Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Msg("request completed")
	}
}
