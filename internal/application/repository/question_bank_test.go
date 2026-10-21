package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestQuestionBankRepositoryScopesBanksQuestionsAndAttempts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:question_bank_repository_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&types.QuestionBank{}, &types.QuestionBankQuestion{}, &types.QuestionBankAttempt{}); err != nil {
		t.Fatal(err)
	}
	repo := NewQuestionBankRepository(db)
	ctx := context.Background()
	now := time.Now()
	bank := &types.QuestionBank{
		ID: "bank-one", TenantID: 1, Name: "Basics", Status: types.QuestionBankStatusPublished,
		SourceKnowledgeBaseIDs: types.JSON(`[]`), CreatedAt: now, UpdatedAt: now,
	}
	otherBank := &types.QuestionBank{
		ID: "bank-two", TenantID: 2, Name: "Other", Status: types.QuestionBankStatusPublished,
		SourceKnowledgeBaseIDs: types.JSON(`[]`), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateBank(ctx, bank); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateBank(ctx, otherBank); err != nil {
		t.Fatal(err)
	}
	visible, err := repo.ListBanks(ctx, 1, true)
	if err != nil || len(visible) != 1 || visible[0].ID != bank.ID {
		t.Fatalf("tenant 1 banks = %#v, err = %v", visible, err)
	}
	if hidden, err := repo.GetBank(ctx, 1, otherBank.ID); err != nil || hidden != nil {
		t.Fatalf("cross-tenant bank lookup = %#v, err = %v", hidden, err)
	}
	question := &types.QuestionBankQuestion{
		ID: "question-one", TenantID: 1, BankID: bank.ID, Type: types.QuestionBankQuestionSingleChoice,
		Prompt: "Which?", Options: types.JSON(`["A","B"]`), CorrectOptions: types.JSON(`[1]`),
		ReferenceAnswer: "B", SourceKnowledgeBaseID: "kb-one",
	}
	if err := db.Create(question).Error; err != nil {
		t.Fatal(err)
	}
	questions, err := repo.ListQuestions(ctx, 1, bank.ID)
	if err != nil || len(questions) != 1 || questions[0].ID != question.ID {
		t.Fatalf("bank questions = %#v, err = %v", questions, err)
	}
	attempt := &types.QuestionBankAttempt{
		ID: "attempt-one", TenantID: 1, BankID: bank.ID, UserID: "user-one",
		Status: types.QuestionBankAttemptInProgress, QuestionCount: 1,
		Items: types.JSON(`[]`), StartedAt: now,
	}
	if err := repo.CreateAttempt(ctx, attempt); err != nil {
		t.Fatal(err)
	}
	if owned, err := repo.GetAttempt(ctx, 1, attempt.ID, "user-one"); err != nil || owned == nil {
		t.Fatalf("owned attempt = %#v, err = %v", owned, err)
	}
	if foreignAttempt, err := repo.GetAttempt(ctx, 1, attempt.ID, "user-two"); err != nil || foreignAttempt != nil {
		t.Fatalf("another user's attempt = %#v, err = %v", foreignAttempt, err)
	}
	if attempts, err := repo.ListAttempts(ctx, 2, bank.ID, "user-one"); err != nil || len(attempts) != 0 {
		t.Fatalf("cross-tenant history = %#v, err = %v", attempts, err)
	}
}
