package dto

// CreateSessionRequest is the payload for creating a session.
type CreateSessionRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// BudgetResponse exposes budget information in responses.
type BudgetResponse struct {
	TotalTokens int     `json:"total_tokens"`
	UsedTokens  int     `json:"used_tokens"`
	TotalCost   float64 `json:"total_cost"`
	UsedCost    float64 `json:"used_cost"`
}

// SessionResponse represents a session in API responses.
type SessionResponse struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Status    string         `json:"status"`
	Budget    BudgetResponse `json:"budget"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}
