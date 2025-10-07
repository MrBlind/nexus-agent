package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(context.Context, *model.Message) error
	GetBySessionID(context.Context, uuid.UUID) ([]*model.Message, error)
	GetByID(context.Context, uuid.UUID) (*model.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, message *model.Message) error {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return err
	}
	return nil
}

func (r *messageRepository) GetBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*model.Message, error) {
	var messages []*model.Message
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *messageRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	var message model.Message
	if err := r.db.WithContext(ctx).First(&message, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &message, nil
}
