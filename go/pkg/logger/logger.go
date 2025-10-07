package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger for consistent logging across the application.
type Logger struct {
	zerolog.Logger
}

// New creates a new logger with the provided log level and writers.
func New(level string, writers ...io.Writer) Logger {
	zerolog.SetGlobalLevel(parseLevel(level))
	writer := io.MultiWriter(writers...)
	if len(writers) == 0 {
		writer = os.Stdout
	}

	return Logger{Logger: zerolog.New(writer).With().Timestamp().Logger()}
}

// NewWithRotation creates a new logger with file rotation support.
func NewWithRotation(level string, logDir string, serviceName string, maxSizeMB int) (Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return Logger{}, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 创建日志文件路径
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", serviceName))
	
	// 打开日志文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return Logger{}, fmt.Errorf("failed to open log file: %w", err)
	}

	// 检查文件大小，如果超过限制则轮转
	if err := rotateIfNeeded(file, logFile, maxSizeMB); err != nil {
		return Logger{}, fmt.Errorf("failed to rotate log file: %w", err)
	}

	// 创建多重写入器：同时输出到控制台和文件
	writer := io.MultiWriter(os.Stdout, file)
	
	zerolog.SetGlobalLevel(parseLevel(level))
	return Logger{Logger: zerolog.New(writer).With().Timestamp().Logger()}, nil
}

// rotateIfNeeded 检查日志文件大小，如果需要则进行轮转
func rotateIfNeeded(file *os.File, logFile string, maxSizeMB int) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// 如果文件大小超过限制，进行轮转
	maxSize := int64(maxSizeMB * 1024 * 1024) // 转换为字节
	if stat.Size() > maxSize {
		// 关闭当前文件
		file.Close()
		
		// 创建备份文件名（带时间戳）
		timestamp := time.Now().Format("20060102-150405")
		backupFile := fmt.Sprintf("%s.%s", logFile, timestamp)
		
		// 重命名当前文件为备份文件
		if err := os.Rename(logFile, backupFile); err != nil {
			return fmt.Errorf("failed to rename log file: %w", err)
		}
		
		// 创建新的日志文件
		newFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to create new log file: %w", err)
		}
		
		// 更新文件句柄
		*file = *newFile
	}
	
	return nil
}

// WithRequestID returns a logger enriched with the request ID field.
func (l Logger) WithRequestID(requestID string) Logger {
	return Logger{Logger: l.Logger.With().Str("request_id", requestID).Logger()}
}

func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
