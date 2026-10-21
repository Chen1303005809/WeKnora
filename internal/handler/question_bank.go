package handler

import (
	"errors"
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// QuestionBankHandler exposes administrator-managed question banks and
// member-owned quiz attempts.
type QuestionBankHandler struct {
	service interfaces.QuestionBankService
}

func NewQuestionBankHandler(service interfaces.QuestionBankService) *QuestionBankHandler {
	return &QuestionBankHandler{service: service}
}

func (h *QuestionBankHandler) ListPublished(c *gin.Context) {
	banks, err := h.service.ListBanks(c.Request.Context(), true)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": banks})
}

func (h *QuestionBankHandler) ListManage(c *gin.Context) {
	banks, err := h.service.ListBanks(c.Request.Context(), false)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": banks})
}

func (h *QuestionBankHandler) GetDraft(c *gin.Context) {
	bank, questions, err := h.service.GetDraftBank(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	if bank == nil {
		c.Error(apperrors.NewNotFoundError("question bank not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"bank": gin.H{
			"id": bank.ID, "name": bank.Name, "description": bank.Description,
			"attempt_question_count": bank.AttemptQuestionCount, "status": bank.Status,
			"grading_prompt": bank.GradingPrompt,
		},
		"questions": questions,
	}})
}

func (h *QuestionBankHandler) Create(c *gin.Context) {
	var req types.QuestionBankCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid question bank request").WithDetails(err.Error()))
		return
	}
	userID, ok := types.UserIDFromContext(c.Request.Context())
	if !ok || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user identity is missing"))
		return
	}
	bank, err := h.service.CreateBank(c.Request.Context(), userID, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": bank})
}

func (h *QuestionBankHandler) UpdateBank(c *gin.Context) {
	var req types.QuestionBankUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid question bank update").WithDetails(err.Error()))
		return
	}
	bank, err := h.service.UpdateBank(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bank})
}

func (h *QuestionBankHandler) UpdateQuestion(c *gin.Context) {
	var req types.QuestionBankQuestionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid question update").WithDetails(err.Error()))
		return
	}
	question, err := h.service.UpdateQuestion(c.Request.Context(), c.Param("id"), c.Param("questionID"), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": question})
}

func (h *QuestionBankHandler) Publish(c *gin.Context) {
	bank, err := h.service.PublishBank(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bank})
}

func (h *QuestionBankHandler) StartAttempt(c *gin.Context) {
	userID, ok := types.UserIDFromContext(c.Request.Context())
	if !ok || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user identity is missing"))
		return
	}
	attempt, err := h.service.StartAttempt(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": attempt})
}

func (h *QuestionBankHandler) SubmitAttempt(c *gin.Context) {
	var req struct {
		Answers []types.QuestionBankAnswer `json:"answers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid attempt submission").WithDetails(err.Error()))
		return
	}
	userID, ok := types.UserIDFromContext(c.Request.Context())
	if !ok || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user identity is missing"))
		return
	}
	attempt, err := h.service.SubmitAttempt(c.Request.Context(), c.Param("attemptID"), userID, req.Answers)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": attempt})
}

func (h *QuestionBankHandler) ListMyAttempts(c *gin.Context) {
	userID, ok := types.UserIDFromContext(c.Request.Context())
	if !ok || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user identity is missing"))
		return
	}
	attempts, err := h.service.ListMyAttempts(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": attempts})
}

func (h *QuestionBankHandler) GetMyAttempt(c *gin.Context) {
	userID, ok := types.UserIDFromContext(c.Request.Context())
	if !ok || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user identity is missing"))
		return
	}
	attempt, err := h.service.GetMyAttempt(c.Request.Context(), c.Param("attemptID"), userID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": attempt})
}

func (h *QuestionBankHandler) fail(c *gin.Context, err error) {
	var appError *apperrors.AppError
	if errors.As(err, &appError) {
		c.Error(appError)
		return
	}
	c.Error(apperrors.NewBadRequestError(err.Error()))
}
