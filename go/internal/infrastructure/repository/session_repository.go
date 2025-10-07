package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"gorm.io/gorm"
)

type SessionRepository interface {
	Create(context.Context, *model.Session) error
	Get(context.Context, uuid.UUID) (*model.Session, error)
	Update(context.Context, *model.Session) error
	Delete(context.Context, uuid.UUID) error
	GetSessions(context.Context) ([]*model.Session, error)

	UpdateBudget(context.Context, uuid.UUID, int, float64) error
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *model.Session) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return err
	}
	return nil
}

func (r *sessionRepository) Get(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	var session model.Session
	if err := r.db.WithContext(ctx).First(&session, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) GetSessions(ctx context.Context) ([]*model.Session, error) {
	var sessions []*model.Session
	if err := r.db.WithContext(ctx).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *sessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&model.Session{}, "id = ?", id).Error; err != nil {
		return err
	}
	return nil
}

func (r *sessionRepository) Update(ctx context.Context, session *model.Session) error {
	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		return err
	}
	return nil
}

func (r *sessionRepository) UpdateBudget(ctx context.Context, id uuid.UUID, tokensUsed int, costUsed float64) error {
	if err := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"budget_used_tokens": gorm.Expr("budget_used_tokens + ?", tokensUsed),
			"budget_used_cost":   gorm.Expr("budget_used_cost + ?", costUsed),
		}).Error; err != nil {
		return err
	}
	return nil
}
