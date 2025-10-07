package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/infrastructure/repository"
)

type MessageService struct {
	repo repository.MessageRepository
}

func NewMessageService(repo repository.MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}

func (s *MessageService) SaveMessage(ctx context.Context, message *model.Message) error {
	return s.repo.Create(ctx, message)
}

func (s *MessageService) SaveMessages(ctx context.Context, messages []*model.Message) error {
	for _, message := range messages {
		if err := s.SaveMessage(ctx, message); err != nil {
			return err
		}
	}
	return nil
}

func (s *MessageService) GetSessionMessages(ctx context.Context, sessionID uuid.UUID) ([]*model.Message, error) {
	return s.repo.GetBySessionID(ctx, sessionID)
}
