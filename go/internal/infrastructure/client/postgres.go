package client

import (
	"fmt"

	"github.com/mrblind/nexus-agent/internal/config"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresClient wraps the gorm DB instance.
type PostgresClient struct {
	DB *gorm.DB
}

// NewPostgresClient creates a new database client using the provided config.
func NewPostgresClient(cfg config.DatabaseConfig, debug bool) (*PostgresClient, error) {
	logLevel := logger.Warn
	// if debug {
	// 	logLevel = logger.Info
	// }

	dsn := cfg.DSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresClient{DB: db}, nil
}

// AutoMigrate runs database migrations for all models.
func (c *PostgresClient) AutoMigrate() error {
	// 自动迁移所有模型
	err := c.DB.AutoMigrate(
		&model.Session{},
		&model.Message{},
		&model.ExecutionTrace{},
		&model.ExecutionStep{},
		&model.Trace{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto migrate database: %w", err)
	}

	fmt.Println("✅ 数据库表自动创建/更新完成")
	return nil
}
