package repository

import (
	"context"

	"github.com/mrblind/nexus-agent/internal/domain/model"
	"gorm.io/gorm"
)

type StepRepository interface {
	Create(ctx context.Context, step *model.ExecutionStep) error
	GetByTraceID(ctx context.Context, traceID string) ([]*model.ExecutionStep, error)
	GetByID(ctx context.Context, stepID string) (*model.ExecutionStep, error)
	Update(ctx context.Context, step *model.ExecutionStep) error
}

type stepRepository struct {
	db *gorm.DB
}

func NewStepRepository(db *gorm.DB) StepRepository {
	return &stepRepository{db: db}
}

func (r *stepRepository) Create(ctx context.Context, step *model.ExecutionStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *stepRepository) GetByTraceID(ctx context.Context, traceID string) ([]*model.ExecutionStep, error) {
	var steps []*model.ExecutionStep
	if err := r.db.WithContext(ctx).Where("trace_id = ?", traceID).Order("sequence ASC").Find(&steps).Error; err != nil {
		return nil, err
	}
	return steps, nil
}

func (r *stepRepository) GetByID(ctx context.Context, stepID string) (*model.ExecutionStep, error) {
	var step model.ExecutionStep
	if err := r.db.WithContext(ctx).First(&step, "id = ?", stepID).Error; err != nil {
		return nil, err
	}
	return &step, nil
}

func (r *stepRepository) Update(ctx context.Context, step *model.ExecutionStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}
