package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	QuestionBankStatusGenerating = "generating"
	QuestionBankStatusDraft      = "draft"
	QuestionBankStatusFailed     = "failed"
	QuestionBankStatusPublished  = "published"

	QuestionBankQuestionSingleChoice   = "single_choice"
	QuestionBankQuestionMultipleChoice = "multiple_choice"
	QuestionBankQuestionShortAnswer    = "short_answer"

	QuestionBankAttemptInProgress = "in_progress"
	QuestionBankAttemptGrading    = "grading"
	QuestionBankAttemptSubmitted  = "submitted"
)

// QuestionBank stores a tenant-shared set of questions and its quiz settings.
// Source IDs and model identifiers are retained for audit and regeneration;
// only administrators receive those details through the API.
type QuestionBank struct {
	ID                     string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID               uint64    `json:"tenant_id" gorm:"not null;index"`
	Name                   string    `json:"name" gorm:"type:varchar(255);not null"`
	Description            string    `json:"description,omitempty" gorm:"type:text"`
	SourceKnowledgeBaseIDs JSON      `json:"-" gorm:"type:jsonb;not null"`
	SingleChoiceCount      int       `json:"single_choice_count" gorm:"not null;default:5"`
	MultipleChoiceCount    int       `json:"multiple_choice_count" gorm:"not null;default:3"`
	ShortAnswerCount       int       `json:"short_answer_count" gorm:"not null;default:2"`
	Difficulty             string    `json:"difficulty,omitempty" gorm:"type:varchar(24);not null;default:''"`
	AttemptQuestionCount   int       `json:"attempt_question_count" gorm:"not null;default:10"`
	GradingPrompt          string    `json:"-" gorm:"type:text;not null"`
	Status                 string    `json:"status" gorm:"type:varchar(24);not null;index"`
	GenerationError        string    `json:"generation_error,omitempty" gorm:"type:text"`
	GenerationModelID      string    `json:"-" gorm:"type:varchar(64);not null;default:''"`
	CreatorID              string    `json:"creator_id,omitempty" gorm:"type:varchar(512);not null;default:'';index"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (QuestionBank) TableName() string { return "question_banks" }

func (b *QuestionBank) BeforeCreate(_ *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	if len(b.SourceKnowledgeBaseIDs) == 0 {
		b.SourceKnowledgeBaseIDs = JSON("[]")
	}
	return nil
}

// QuestionBankQuestion is one generated and administrator-editable question.
// Options and correct option indexes are JSON arrays; short-answer questions
// leave both arrays empty and use ReferenceAnswer.
type QuestionBankQuestion struct {
	ID                      string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID                uint64    `json:"-" gorm:"not null;index"`
	BankID                  string    `json:"bank_id" gorm:"type:varchar(36);not null;index"`
	Type                    string    `json:"type" gorm:"type:varchar(24);not null"`
	Prompt                  string    `json:"prompt" gorm:"type:text;not null"`
	Options                 JSON      `json:"options" gorm:"type:jsonb;not null"`
	CorrectOptions          JSON      `json:"correct_options,omitempty" gorm:"type:jsonb;not null"`
	ReferenceAnswer         string    `json:"reference_answer" gorm:"type:text;not null"`
	Explanation             string    `json:"explanation,omitempty" gorm:"type:text"`
	SourceKnowledgeBaseID   string    `json:"source_knowledge_base_id,omitempty" gorm:"type:varchar(36);not null;default:''"`
	SourceKnowledgeBaseName string    `json:"source_knowledge_base_name,omitempty" gorm:"type:varchar(255);not null;default:''"`
	SourceKnowledgeID       string    `json:"source_knowledge_id,omitempty" gorm:"type:varchar(36);not null;default:''"`
	SourceChunkID           string    `json:"source_chunk_id,omitempty" gorm:"type:varchar(36);not null;default:''"`
	SourceTitle             string    `json:"source_title,omitempty" gorm:"type:varchar(512);not null;default:''"`
	SortOrder               int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (QuestionBankQuestion) TableName() string { return "question_bank_questions" }

func (q *QuestionBankQuestion) BeforeCreate(_ *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.NewString()
	}
	if len(q.Options) == 0 {
		q.Options = JSON("[]")
	}
	if len(q.CorrectOptions) == 0 {
		q.CorrectOptions = JSON("[]")
	}
	return nil
}

// QuestionBankAttempt persists a learner's selected question snapshots,
// submitted answers and final feedback. Items and grading instructions stay
// out of the default JSON representation and are returned through purpose-built
// DTOs after ownership checks.
type QuestionBankAttempt struct {
	ID                    string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID              uint64     `json:"-" gorm:"not null;index"`
	BankID                string     `json:"bank_id" gorm:"type:varchar(36);not null;index"`
	UserID                string     `json:"-" gorm:"type:varchar(512);not null;index"`
	Status                string     `json:"status" gorm:"type:varchar(24);not null;index"`
	QuestionCount         int        `json:"question_count" gorm:"not null;default:0"`
	Score                 float64    `json:"score" gorm:"not null;default:0"`
	CorrectCount          int        `json:"correct_count" gorm:"not null;default:0"`
	Items                 JSON       `json:"-" gorm:"type:jsonb;not null"`
	GradingPromptSnapshot string     `json:"-" gorm:"type:text;not null"`
	ModelID               string     `json:"model_id,omitempty" gorm:"type:varchar(64);not null;default:''"`
	StartedAt             time.Time  `json:"started_at"`
	SubmittedAt           *time.Time `json:"submitted_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (QuestionBankAttempt) TableName() string { return "question_bank_attempts" }

func (a *QuestionBankAttempt) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if len(a.Items) == 0 {
		a.Items = JSON("[]")
	}
	return nil
}

type QuestionBankCreateRequest struct {
	Name                   string   `json:"name"`
	Description            string   `json:"description,omitempty"`
	SourceKnowledgeBaseIDs []string `json:"source_knowledge_base_ids"`
	SingleChoiceCount      int      `json:"single_choice_count"`
	MultipleChoiceCount    int      `json:"multiple_choice_count"`
	ShortAnswerCount       int      `json:"short_answer_count"`
	Difficulty             string   `json:"difficulty,omitempty"`
	AttemptQuestionCount   int      `json:"attempt_question_count"`
	GradingPrompt          string   `json:"grading_prompt"`
}

type QuestionBankUpdateRequest struct {
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	AttemptQuestionCount int    `json:"attempt_question_count"`
	GradingPrompt        string `json:"grading_prompt"`
}

type QuestionBankQuestionUpdateRequest struct {
	Prompt          string   `json:"prompt"`
	Options         []string `json:"options,omitempty"`
	CorrectOptions  []int    `json:"correct_options,omitempty"`
	ReferenceAnswer string   `json:"reference_answer"`
	Explanation     string   `json:"explanation,omitempty"`
}

type QuestionBankAnswer struct {
	QuestionID      string `json:"question_id"`
	SelectedOptions []int  `json:"selected_options,omitempty"`
	Text            string `json:"text,omitempty"`
}

// QuestionBankAttemptItem is the durable snapshot used to grade and review an
// attempt even if the underlying question bank is later edited.
type QuestionBankAttemptItem struct {
	Question        QuestionBankQuestion `json:"question"`
	SelectedOptions []int                `json:"selected_options,omitempty"`
	TextAnswer      string               `json:"text_answer,omitempty"`
	Score           float64              `json:"score,omitempty"`
	Correct         *bool                `json:"correct,omitempty"`
	Feedback        string               `json:"feedback,omitempty"`
}

type QuestionBankAttemptQuestion struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options,omitempty"`
}

type QuestionBankAttemptResultItem struct {
	ID                    string   `json:"id"`
	Type                  string   `json:"type"`
	Prompt                string   `json:"prompt"`
	Options               []string `json:"options,omitempty"`
	SelectedOptions       []int    `json:"selected_options,omitempty"`
	TextAnswer            string   `json:"text_answer,omitempty"`
	CorrectOptions        []int    `json:"correct_options,omitempty"`
	ReferenceAnswer       string   `json:"reference_answer"`
	Explanation           string   `json:"explanation,omitempty"`
	Score                 float64  `json:"score"`
	Correct               *bool    `json:"correct,omitempty"`
	Feedback              string   `json:"feedback,omitempty"`
	SourceKnowledgeBaseID string   `json:"source_knowledge_base_id,omitempty"`
	SourceTitle           string   `json:"source_title,omitempty"`
	SourceAccessible      bool     `json:"source_accessible"`
}

type QuestionBankAttemptResponse struct {
	Attempt   *QuestionBankAttempt            `json:"attempt"`
	Questions []QuestionBankAttemptQuestion   `json:"questions,omitempty"`
	Items     []QuestionBankAttemptResultItem `json:"items,omitempty"`
}
