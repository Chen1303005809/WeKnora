package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type QuestionBankRepository interface {
	CreateBank(ctx context.Context, bank *types.QuestionBank) error
	GetBank(ctx context.Context, tenantID uint64, bankID string) (*types.QuestionBank, error)
	ListBanks(ctx context.Context, tenantID uint64, publishedOnly bool) ([]*types.QuestionBank, error)
	UpdateBank(ctx context.Context, bank *types.QuestionBank) error
	ReplaceQuestions(ctx context.Context, tenantID uint64, bankID string, questions []*types.QuestionBankQuestion) error
	ListQuestions(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionBankQuestion, error)
	GetQuestion(ctx context.Context, tenantID uint64, bankID, questionID string) (*types.QuestionBankQuestion, error)
	UpdateQuestion(ctx context.Context, question *types.QuestionBankQuestion) error
	CreateAttempt(ctx context.Context, attempt *types.QuestionBankAttempt) error
	GetAttempt(ctx context.Context, tenantID uint64, attemptID, userID string) (*types.QuestionBankAttempt, error)
	ListAttempts(ctx context.Context, tenantID uint64, bankID, userID string) ([]*types.QuestionBankAttempt, error)
	ClaimAttemptForGrading(ctx context.Context, tenantID uint64, attemptID, userID string) (bool, error)
	UpdateAttempt(ctx context.Context, attempt *types.QuestionBankAttempt) error
}

type QuestionBankService interface {
	ListBanks(ctx context.Context, publishedOnly bool) ([]*types.QuestionBank, error)
	GetDraftBank(ctx context.Context, bankID string) (*types.QuestionBank, []*types.QuestionBankQuestion, error)
	CreateBank(ctx context.Context, creatorID string, req types.QuestionBankCreateRequest) (*types.QuestionBank, error)
	UpdateBank(ctx context.Context, bankID string, req types.QuestionBankUpdateRequest) (*types.QuestionBank, error)
	UpdateQuestion(ctx context.Context, bankID, questionID string, req types.QuestionBankQuestionUpdateRequest) (*types.QuestionBankQuestion, error)
	PublishBank(ctx context.Context, bankID string) (*types.QuestionBank, error)
	StartAttempt(ctx context.Context, bankID, userID string) (*types.QuestionBankAttemptResponse, error)
	SubmitAttempt(ctx context.Context, attemptID, userID string, answers []types.QuestionBankAnswer) (*types.QuestionBankAttemptResponse, error)
	ListMyAttempts(ctx context.Context, bankID, userID string) ([]*types.QuestionBankAttempt, error)
	GetMyAttempt(ctx context.Context, attemptID, userID string) (*types.QuestionBankAttemptResponse, error)
	HandleGeneration(ctx context.Context, task *asynq.Task) error
}
