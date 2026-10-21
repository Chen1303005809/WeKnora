package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type questionBankRepository struct {
	db *gorm.DB
}

func NewQuestionBankRepository(db *gorm.DB) interfaces.QuestionBankRepository {
	return &questionBankRepository{db: db}
}

func (r *questionBankRepository) CreateBank(ctx context.Context, bank *types.QuestionBank) error {
	return r.db.WithContext(ctx).Create(bank).Error
}

func (r *questionBankRepository) GetBank(
	ctx context.Context, tenantID uint64, bankID string,
) (*types.QuestionBank, error) {
	var bank types.QuestionBank
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, bankID).
		First(&bank).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &bank, nil
}

func (r *questionBankRepository) ListBanks(
	ctx context.Context, tenantID uint64, publishedOnly bool,
) ([]*types.QuestionBank, error) {
	var banks []*types.QuestionBank
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if publishedOnly {
		query = query.Where("status = ?", types.QuestionBankStatusPublished)
	}
	err := query.Order("updated_at DESC").Order("id DESC").Find(&banks).Error
	return banks, err
}

func (r *questionBankRepository) UpdateBank(ctx context.Context, bank *types.QuestionBank) error {
	if bank == nil {
		return errors.New("question bank is nil")
	}
	result := r.db.WithContext(ctx).Model(&types.QuestionBank{}).
		Where("tenant_id = ? AND id = ?", bank.TenantID, bank.ID).
		Select("*").Updates(bank)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *questionBankRepository) ReplaceQuestions(
	ctx context.Context,
	tenantID uint64,
	bankID string,
	questions []*types.QuestionBankQuestion,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND bank_id = ?", tenantID, bankID).
			Delete(&types.QuestionBankQuestion{}).Error; err != nil {
			return err
		}
		if len(questions) == 0 {
			return nil
		}
		return tx.Create(&questions).Error
	})
}

func (r *questionBankRepository) ListQuestions(
	ctx context.Context, tenantID uint64, bankID string,
) ([]*types.QuestionBankQuestion, error) {
	var questions []*types.QuestionBankQuestion
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND bank_id = ?", tenantID, bankID).
		Order("sort_order ASC").Order("id ASC").Find(&questions).Error
	return questions, err
}

func (r *questionBankRepository) GetQuestion(
	ctx context.Context, tenantID uint64, bankID, questionID string,
) (*types.QuestionBankQuestion, error) {
	var question types.QuestionBankQuestion
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND bank_id = ? AND id = ?", tenantID, bankID, questionID).
		First(&question).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &question, nil
}

func (r *questionBankRepository) UpdateQuestion(ctx context.Context, question *types.QuestionBankQuestion) error {
	if question == nil {
		return errors.New("question is nil")
	}
	result := r.db.WithContext(ctx).Model(&types.QuestionBankQuestion{}).
		Where("tenant_id = ? AND bank_id = ? AND id = ?", question.TenantID, question.BankID, question.ID).
		Select("*").Updates(question)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *questionBankRepository) CreateAttempt(ctx context.Context, attempt *types.QuestionBankAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *questionBankRepository) GetAttempt(
	ctx context.Context, tenantID uint64, attemptID, userID string,
) (*types.QuestionBankAttempt, error) {
	var attempt types.QuestionBankAttempt
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ? AND user_id = ?", tenantID, attemptID, userID).
		First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attempt, nil
}

func (r *questionBankRepository) ListAttempts(
	ctx context.Context, tenantID uint64, bankID, userID string,
) ([]*types.QuestionBankAttempt, error) {
	var attempts []*types.QuestionBankAttempt
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND bank_id = ? AND user_id = ?", tenantID, bankID, userID).
		Order("started_at DESC").Order("id DESC").Find(&attempts).Error
	return attempts, err
}

func (r *questionBankRepository) ClaimAttemptForGrading(
	ctx context.Context, tenantID uint64, attemptID, userID string,
) (bool, error) {
	result := r.db.WithContext(ctx).Model(&types.QuestionBankAttempt{}).
		Where("tenant_id = ? AND id = ? AND user_id = ? AND status = ?",
			tenantID, attemptID, userID, types.QuestionBankAttemptInProgress).
		Updates(map[string]interface{}{
			"status":     types.QuestionBankAttemptGrading,
			"updated_at": time.Now(),
		})
	return result.RowsAffected == 1, result.Error
}

func (r *questionBankRepository) UpdateAttempt(ctx context.Context, attempt *types.QuestionBankAttempt) error {
	if attempt == nil {
		return errors.New("question bank attempt is nil")
	}
	result := r.db.WithContext(ctx).Model(&types.QuestionBankAttempt{}).
		Where("tenant_id = ? AND id = ? AND user_id = ?", attempt.TenantID, attempt.ID, attempt.UserID).
		Select("*").Updates(attempt)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
