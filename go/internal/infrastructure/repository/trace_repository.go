package repository

import (
	"context"
	"log"
	"time"

	"github.com/mrblind/nexus-agent/internal/domain/model"
	"gorm.io/gorm"
)

// Repository接口定义
type TraceRepository interface {
	Create(ctx context.Context, trace *model.Trace) error
	GetByID(ctx context.Context, traceID string) (*model.Trace, error)
	GetBySessionID(ctx context.Context, sessionID string) ([]*model.Trace, error) // 跨表查询
	UpdateStatus(ctx context.Context, traceID string, status model.TraceStatus, endedAt time.Time) error
	UpdateStatusWithError(ctx context.Context, traceID string, status model.TraceStatus, endedAt time.Time, err error) error
	UpdateCost(ctx context.Context, traceID string, tokens int, apiCost float64) error
}

type traceRepository struct {
	db *gorm.DB
}

func NewTraceRepository(db *gorm.DB) TraceRepository {
	return &traceRepository{db: db}
}

func (r *traceRepository) Create(ctx context.Context, trace *model.Trace) error {
	return r.db.WithContext(ctx).Create(trace).Error
}

func (r *traceRepository) GetByID(ctx context.Context, traceID string) (*model.Trace, error) {
	var trace model.Trace
	if err := r.db.WithContext(ctx).First(&trace, "id = ?", traceID).Error; err != nil {
		return nil, err
	}
	return &trace, nil
}

func (r *traceRepository) UpdateStatus(ctx context.Context, traceID string, status model.TraceStatus, endedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&model.Trace{}).Where("id = ?", traceID).Updates(map[string]interface{}{
		"status":   status,
		"ended_at": endedAt,
	}).Error
}

func (r *traceRepository) UpdateStatusWithError(ctx context.Context, traceID string, status model.TraceStatus, endedAt time.Time, err error) error {

	return r.db.WithContext(ctx).Model(&model.Trace{}).Where("id = ?", traceID).Updates(map[string]interface{}{
		"status":   status,
		"ended_at": endedAt,
		"error":    err.Error(),
	}).Error
}

func (r *traceRepository) UpdateCost(ctx context.Context, traceID string, tokens int, apiCost float64) error {
	// 使用原生SQL进行累加操作
	log.Printf("UpdateCost - TraceID: %s, Tokens: %d, ApiCost: %f", traceID, tokens, apiCost)

	result := r.db.WithContext(ctx).Exec(`
		UPDATE traces 
		SET cost_tokens = COALESCE(cost_tokens, 0) + ?, 
		    cost_api = COALESCE(cost_api, 0) + ?
		WHERE id = ?
	`, tokens, apiCost, traceID)

	log.Printf("UpdateCost结果 - RowsAffected: %d, Error: %v", result.RowsAffected, result.Error)
	return result.Error
}

// GetBySessionID 获取session下的所有traces（跨表查询实现）
func (r *traceRepository) GetBySessionID(ctx context.Context, sessionID string) ([]*model.Trace, error) {
	var traces []*model.Trace
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC"). // 最新的在前
		Find(&traces).Error
	return traces, err
}
