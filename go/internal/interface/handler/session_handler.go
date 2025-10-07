package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mrblind/nexus-agent/internal/domain/model"
	"github.com/mrblind/nexus-agent/internal/domain/service"
	"github.com/mrblind/nexus-agent/internal/interface/dto"
	"github.com/mrblind/nexus-agent/internal/interface/middleware"
)

type SessionHandler struct {
	service *service.SessionService
}

func NewSessionHandler(service *service.SessionService) SessionHandler {
	return SessionHandler{
		service: service,
	}
}

func (h SessionHandler) Create(c *gin.Context) {
	log := middleware.FromContext(c.Request.Context())

	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("创建会话请求参数无效")
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_INPUT", "message": err.Error()})
		return
	}

	log.Info().Str("user_id", req.UserID).Msg("创建新会话")

	session, err := h.service.Create(c.Request.Context(), req.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("创建会话失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CREATE_SESSION_FAILED", "message": err.Error()})
		return
	}

	log.Info().
		Str("session_id", session.ID.String()).
		Str("user_id", req.UserID).
		Msg("会话创建成功")

	c.JSON(http.StatusCreated, toSessionResponse(session))
}

func (h SessionHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_SESSION_ID"})
		return
	}

	session, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SESSION_NOT_FOUND"})
		return
	}

	c.JSON(http.StatusOK, toSessionResponse(session))
}

func (h SessionHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_SESSION_ID"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DELETE_SESSION_FAILED", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted"})
}

func (h SessionHandler) GetList(c *gin.Context) {
	sessions, err := h.service.GetSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GET_SESSIONS_FAILED", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toSessionResponses(sessions))
}

func toSessionResponse(s *model.Session) dto.SessionResponse {
	return dto.SessionResponse{
		ID:     s.ID.String(),
		UserID: s.UserID,
		Status: s.Status,
		Budget: dto.BudgetResponse{
			TotalTokens: s.Budget.TotalTokens,
			UsedTokens:  s.Budget.UsedTokens,
			TotalCost:   s.Budget.TotalCost,
			UsedCost:    s.Budget.UsedCost,
		},
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
}

func toSessionResponses(sessions []*model.Session) []dto.SessionResponse {
	if len(sessions) == 0 {
		return []dto.SessionResponse{}
	}

	responses := make([]dto.SessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = toSessionResponse(session)
	}
	return responses
}
