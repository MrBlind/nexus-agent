package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/config"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/infrastructure/repository"
)

type SessionService struct {
	repo   repository.SessionRepository
	budget config.BudgetConfig
}

func NewSessionService(repo repository.SessionRepository, budget config.BudgetConfig) *SessionService {
	return &SessionService{repo: repo, budget: budget}
}

func (s *SessionService) Create(ctx context.Context, userID string) (*model.Session, error) {
	session := &model.Session{
		ID:     uuid.New(),
		UserID: userID,
		Status: "active",
		Budget: model.Budget{
			TotalTokens: s.budget.DefaultTotalTokens,
			UsedTokens:  0,
			TotalCost:   s.budget.DefaultTotalCost,
			UsedCost:    0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *SessionService) Get(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	return s.repo.Get(ctx, id)
}

func (s *SessionService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *SessionService) GetSessions(ctx context.Context) ([]*model.Session, error) {
	return s.repo.GetSessions(ctx)
}

func (s *SessionService) UpdateBudget(ctx context.Context, id uuid.UUID, tokensUsed int, costUsed float64) error {
	return s.repo.UpdateBudget(ctx, id, tokensUsed, costUsed)
}

func (s *SessionService) CheckBudget(ctx context.Context, session *model.Session, addtionalTokens int, addtionalCost float64) error {
	if session.Budget.TotalTokens > 0 {
		if session.Budget.UsedTokens+addtionalTokens > session.Budget.TotalTokens {
			return errors.New("token budget exceeded")
		}
	}

	if session.Budget.TotalCost > 0 {
		if session.Budget.UsedCost+addtionalCost > session.Budget.TotalCost {
			return errors.New("cost budget exceeded")
		}
	}

	return nil
}
