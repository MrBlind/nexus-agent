package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/mrblind/nexus-agent/pkg/logger"
	"github.com/mrblind/nexus-agent/pkg/trace"
)

// ContextLoggerKey 是存储日志器的上下文键
const ContextLoggerKey = "logger"

// ContextLogger 中间件将带有 request_id 的日志器注入到上下文中
func ContextLogger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取 request_id
		reqID := trace.FromContext(c.Request.Context())

		// 创建带有 request_id 的日志器
		requestLogger := log.WithRequestID(reqID)

		// 将日志器存储到上下文中
		ctx := context.WithValue(c.Request.Context(), ContextLoggerKey, requestLogger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// FromContext 从上下文中获取日志器
func FromContext(ctx context.Context) logger.Logger {
	if val, ok := ctx.Value(ContextLoggerKey).(logger.Logger); ok {
		return val
	}
	// 如果上下文中没有日志器，返回一个默认的
	return logger.Logger{}
}
