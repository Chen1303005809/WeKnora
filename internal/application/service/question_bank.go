package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/access"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	defaultQuestionBankGradingPrompt = "根据题目、参考答案和评分标准，公平评估学习者的回答。接受语义正确的同义表达；不要求逐字一致。指出回答正确之处和缺少的关键信息。"
	maxQuestionBankQuestionCount     = 50
)

type questionBankService struct {
	repo         interfaces.QuestionBankRepository
	kbService    interfaces.KnowledgeBaseService
	kbShare      interfaces.KBShareService
	knowledge    interfaces.KnowledgeService
	chunks       interfaces.ChunkService
	models       interfaces.ModelService
	taskEnqueuer interfaces.TaskEnqueuer
}

func NewQuestionBankService(
	repo interfaces.QuestionBankRepository,
	kbService interfaces.KnowledgeBaseService,
	kbShare interfaces.KBShareService,
	knowledge interfaces.KnowledgeService,
	chunks interfaces.ChunkService,
	models interfaces.ModelService,
	taskEnqueuer interfaces.TaskEnqueuer,
) interfaces.QuestionBankService {
	return &questionBankService{
		repo: repo, kbService: kbService, kbShare: kbShare,
		knowledge: knowledge, chunks: chunks, models: models, taskEnqueuer: taskEnqueuer,
	}
}

func (s *questionBankService) ListBanks(
	ctx context.Context, publishedOnly bool,
) ([]*types.QuestionBank, error) {
	return s.repo.ListBanks(ctx, types.MustTenantIDFromContext(ctx), publishedOnly)
}

func (s *questionBankService) GetDraftBank(
	ctx context.Context, bankID string,
) (*types.QuestionBank, []*types.QuestionBankQuestion, error) {
	bank, err := s.getBank(ctx, bankID)
	if err != nil {
		return nil, nil, err
	}
	questions, err := s.repo.ListQuestions(ctx, bank.TenantID, bank.ID)
	return bank, questions, err
}

func (s *questionBankService) CreateBank(
	ctx context.Context, creatorID string, req types.QuestionBankCreateRequest,
) (*types.QuestionBank, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("question bank name is required")
	}
	if len([]rune(req.Name)) > 255 {
		return nil, errors.New("question bank name must be 255 characters or fewer")
	}
	ids := uniqueNonEmpty(req.SourceKnowledgeBaseIDs)
	if len(ids) == 0 {
		return nil, errors.New("select at least one knowledge base")
	}
	if req.SingleChoiceCount == 0 && req.MultipleChoiceCount == 0 && req.ShortAnswerCount == 0 {
		req.SingleChoiceCount, req.MultipleChoiceCount, req.ShortAnswerCount = 5, 3, 2
	}
	total := req.SingleChoiceCount + req.MultipleChoiceCount + req.ShortAnswerCount
	if req.SingleChoiceCount < 0 || req.MultipleChoiceCount < 0 || req.ShortAnswerCount < 0 ||
		req.SingleChoiceCount > maxQuestionBankQuestionCount || req.MultipleChoiceCount > maxQuestionBankQuestionCount ||
		req.ShortAnswerCount > maxQuestionBankQuestionCount || total <= 0 || total > maxQuestionBankQuestionCount {
		return nil, fmt.Errorf("total generated questions must be between 1 and %d", maxQuestionBankQuestionCount)
	}
	if req.AttemptQuestionCount <= 0 {
		req.AttemptQuestionCount = 10
	} else if req.AttemptQuestionCount > maxQuestionBankQuestionCount {
		return nil, fmt.Errorf("attempt question count must be %d or fewer", maxQuestionBankQuestionCount)
	}
	if !validDifficulty(req.Difficulty) {
		return nil, errors.New("difficulty must be easy, medium, hard, or mixed")
	}
	if strings.TrimSpace(req.GradingPrompt) == "" {
		req.GradingPrompt = defaultQuestionBankGradingPrompt
	}

	// A tenant administrator may create banks only from KBs that they can
	// actually read, including organization-shared KBs. Published banks are a
	// separate, explicit tenant-wide copy of the generated Q&A content.
	kbs, err := s.kbService.GetKnowledgeBasesByIDsOnly(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(kbs) != len(ids) {
		return nil, errors.New("one or more selected knowledge bases could not be found")
	}
	if err := access.AuthorizeKBAccess(ctx, s.kbShare, kbs); err != nil {
		return nil, err
	}
	if _, _, err := s.defaultModel(ctx); err != nil {
		return nil, err
	}

	sourceIDsJSON, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	bank := &types.QuestionBank{
		ID: uuid.NewString(), TenantID: types.MustTenantIDFromContext(ctx),
		Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description),
		SourceKnowledgeBaseIDs: types.JSON(sourceIDsJSON),
		SingleChoiceCount:      req.SingleChoiceCount, MultipleChoiceCount: req.MultipleChoiceCount,
		ShortAnswerCount: req.ShortAnswerCount, Difficulty: req.Difficulty,
		AttemptQuestionCount: req.AttemptQuestionCount, GradingPrompt: strings.TrimSpace(req.GradingPrompt),
		Status: types.QuestionBankStatusGenerating, CreatorID: creatorID,
	}
	if err := s.repo.CreateBank(ctx, bank); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(questionBankGenerationPayload{
		TenantID: bank.TenantID, UserID: creatorID, BankID: bank.ID,
	})
	if err != nil {
		return nil, err
	}
	if s.taskEnqueuer == nil {
		_ = s.failGeneration(ctx, bank, errors.New("question generation task queue is unavailable"))
		return s.repo.GetBank(ctx, bank.TenantID, bank.ID)
	}
	task := asynq.NewTask(types.TypeQuestionBankGeneration, payload,
		asynq.Queue(types.QueueQuestion), asynq.MaxRetry(1), asynq.Timeout(20*time.Minute))
	if _, err := s.taskEnqueuer.Enqueue(task); err != nil {
		if saveErr := s.failGeneration(ctx, bank, fmt.Errorf("enqueue question generation: %w", err)); saveErr != nil {
			return nil, saveErr
		}
	}
	// Lite mode may execute the task inline, so return its persisted final state.
	return s.repo.GetBank(ctx, bank.TenantID, bank.ID)
}

func (s *questionBankService) UpdateBank(
	ctx context.Context, bankID string, req types.QuestionBankUpdateRequest,
) (*types.QuestionBank, error) {
	bank, err := s.getBank(ctx, bankID)
	if err != nil {
		return nil, err
	}
	if bank.Status != types.QuestionBankStatusDraft && bank.Status != types.QuestionBankStatusFailed {
		return nil, errors.New("only draft or failed question banks can be edited")
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		if len([]rune(name)) > 255 {
			return nil, errors.New("question bank name must be 255 characters or fewer")
		}
		bank.Name = name
	}
	bank.Description = strings.TrimSpace(req.Description)
	if req.AttemptQuestionCount > 0 {
		if req.AttemptQuestionCount > maxQuestionBankQuestionCount {
			return nil, fmt.Errorf("attempt question count must be %d or fewer", maxQuestionBankQuestionCount)
		}
		bank.AttemptQuestionCount = req.AttemptQuestionCount
	}
	if strings.TrimSpace(req.GradingPrompt) != "" {
		bank.GradingPrompt = strings.TrimSpace(req.GradingPrompt)
	}
	if err := s.repo.UpdateBank(ctx, bank); err != nil {
		return nil, err
	}
	return bank, nil
}

func (s *questionBankService) UpdateQuestion(
	ctx context.Context, bankID, questionID string, req types.QuestionBankQuestionUpdateRequest,
) (*types.QuestionBankQuestion, error) {
	bank, err := s.getBank(ctx, bankID)
	if err != nil {
		return nil, err
	}
	if bank.Status != types.QuestionBankStatusDraft {
		return nil, errors.New("only draft question banks can be edited")
	}
	question, err := s.repo.GetQuestion(ctx, bank.TenantID, bank.ID, questionID)
	if err != nil || question == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("question not found")
	}
	if err := validateQuestionUpdate(question.Type, req); err != nil {
		return nil, err
	}
	question.Prompt = strings.TrimSpace(req.Prompt)
	question.ReferenceAnswer = strings.TrimSpace(req.ReferenceAnswer)
	question.Explanation = strings.TrimSpace(req.Explanation)
	options := req.Options
	if options == nil {
		options = []string{}
	}
	correctOptions := req.CorrectOptions
	if correctOptions == nil {
		correctOptions = []int{}
	}
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	correctJSON, err := json.Marshal(correctOptions)
	if err != nil {
		return nil, err
	}
	question.Options = types.JSON(optionsJSON)
	question.CorrectOptions = types.JSON(correctJSON)
	if err := s.repo.UpdateQuestion(ctx, question); err != nil {
		return nil, err
	}
	return question, nil
}

func (s *questionBankService) PublishBank(
	ctx context.Context, bankID string,
) (*types.QuestionBank, error) {
	bank, err := s.getBank(ctx, bankID)
	if err != nil {
		return nil, err
	}
	if bank.Status != types.QuestionBankStatusDraft {
		return nil, errors.New("only draft question banks can be published")
	}
	questions, err := s.repo.ListQuestions(ctx, bank.TenantID, bank.ID)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, errors.New("question bank must contain questions before publishing")
	}
	bank.Status = types.QuestionBankStatusPublished
	bank.GenerationError = ""
	if err := s.repo.UpdateBank(ctx, bank); err != nil {
		return nil, err
	}
	return bank, nil
}

func (s *questionBankService) StartAttempt(
	ctx context.Context, bankID, userID string,
) (*types.QuestionBankAttemptResponse, error) {
	bank, err := s.getBank(ctx, bankID)
	if err != nil {
		return nil, err
	}
	if bank.Status != types.QuestionBankStatusPublished {
		return nil, errors.New("question bank is not published")
	}
	questions, err := s.repo.ListQuestions(ctx, bank.TenantID, bank.ID)
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, errors.New("question bank has no published questions")
	}
	perm := rand.Perm(len(questions))
	count := bank.AttemptQuestionCount
	if count <= 0 || count > len(questions) {
		count = len(questions)
	}
	items := make([]types.QuestionBankAttemptItem, 0, count)
	publicQuestions := make([]types.QuestionBankAttemptQuestion, 0, count)
	for _, index := range perm[:count] {
		question := *questions[index]
		items = append(items, types.QuestionBankAttemptItem{Question: question})
		publicQuestions = append(publicQuestions, types.QuestionBankAttemptQuestion{
			ID: question.ID, Type: question.Type, Prompt: question.Prompt,
			Options: decodeStringArray(question.Options),
		})
	}
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	attempt := &types.QuestionBankAttempt{
		TenantID: bank.TenantID, BankID: bank.ID, UserID: userID,
		Status: types.QuestionBankAttemptInProgress, QuestionCount: count,
		Items: types.JSON(itemsJSON), GradingPromptSnapshot: bank.GradingPrompt,
		StartedAt: time.Now(),
	}
	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		return nil, err
	}
	return &types.QuestionBankAttemptResponse{Attempt: attempt, Questions: publicQuestions}, nil
}

func (s *questionBankService) SubmitAttempt(
	ctx context.Context, attemptID, userID string, answers []types.QuestionBankAnswer,
) (*types.QuestionBankAttemptResponse, error) {
	tenantID := types.MustTenantIDFromContext(ctx)
	attempt, err := s.repo.GetAttempt(ctx, tenantID, attemptID, userID)
	if err != nil || attempt == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("attempt not found")
	}
	if attempt.Status == types.QuestionBankAttemptSubmitted {
		return s.buildAttemptResponse(ctx, attempt, true)
	}
	if attempt.Status != types.QuestionBankAttemptInProgress {
		return nil, errors.New("attempt is already being graded")
	}
	items, err := decodeAttemptItems(attempt.Items)
	if err != nil {
		return nil, fmt.Errorf("invalid stored attempt: %w", err)
	}
	if err := applyAnswers(items, answers); err != nil {
		return nil, err
	}
	claimed, err := s.repo.ClaimAttemptForGrading(ctx, tenantID, attempt.ID, userID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, errors.New("attempt is already being graded")
	}
	attempt.Status = types.QuestionBankAttemptGrading
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	attempt.Items = types.JSON(itemsJSON)
	if err := s.repo.UpdateAttempt(ctx, attempt); err != nil {
		return nil, err
	}

	modelID := ""
	if err := gradeObjectiveQuestions(items); err != nil {
		return nil, err
	}
	shortAnswerCount := countShortAnswers(items)
	if shortAnswerCount > 0 {
		model, id, modelErr := s.defaultModel(ctx)
		if modelErr != nil {
			return s.releaseAttemptAfterGradingError(ctx, attempt, modelErr)
		}
		modelID = id
		if err := s.gradeShortAnswers(ctx, model, attempt.GradingPromptSnapshot, items); err != nil {
			return s.releaseAttemptAfterGradingError(ctx, attempt, err)
		}
	}
	var total float64
	correctCount := 0
	for i := range items {
		total += items[i].Score
		if items[i].Score >= 100 {
			correctCount++
		}
	}
	attempt.Score = math.Round((total/float64(len(items)))*100) / 100
	attempt.CorrectCount = correctCount
	attempt.ModelID = modelID
	attempt.Status = types.QuestionBankAttemptSubmitted
	now := time.Now()
	attempt.SubmittedAt = &now
	itemsJSON, err = json.Marshal(items)
	if err != nil {
		return nil, err
	}
	attempt.Items = types.JSON(itemsJSON)
	if err := s.repo.UpdateAttempt(ctx, attempt); err != nil {
		return nil, err
	}
	return s.buildAttemptResponse(ctx, attempt, true)
}

func (s *questionBankService) ListMyAttempts(
	ctx context.Context, bankID, userID string,
) ([]*types.QuestionBankAttempt, error) {
	bank, err := s.getBank(ctx, bankID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAttempts(ctx, bank.TenantID, bank.ID, userID)
}

func (s *questionBankService) GetMyAttempt(
	ctx context.Context, attemptID, userID string,
) (*types.QuestionBankAttemptResponse, error) {
	attempt, err := s.repo.GetAttempt(ctx, types.MustTenantIDFromContext(ctx), attemptID, userID)
	if err != nil || attempt == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("attempt not found")
	}
	return s.buildAttemptResponse(ctx, attempt, attempt.Status == types.QuestionBankAttemptSubmitted)
}

func (s *questionBankService) releaseAttemptAfterGradingError(
	ctx context.Context, attempt *types.QuestionBankAttempt, cause error,
) (*types.QuestionBankAttemptResponse, error) {
	attempt.Status = types.QuestionBankAttemptInProgress
	if err := s.repo.UpdateAttempt(ctx, attempt); err != nil {
		return nil, fmt.Errorf("%v; failed to reopen attempt: %w", cause, err)
	}
	return nil, cause
}

func (s *questionBankService) buildAttemptResponse(
	ctx context.Context, attempt *types.QuestionBankAttempt, includeResults bool,
) (*types.QuestionBankAttemptResponse, error) {
	response := &types.QuestionBankAttemptResponse{Attempt: attempt}
	items, err := decodeAttemptItems(attempt.Items)
	if err != nil {
		return nil, err
	}
	if !includeResults {
		response.Questions = make([]types.QuestionBankAttemptQuestion, 0, len(items))
		for _, item := range items {
			response.Questions = append(response.Questions, types.QuestionBankAttemptQuestion{
				ID: item.Question.ID, Type: item.Question.Type, Prompt: item.Question.Prompt,
				Options: decodeStringArray(item.Question.Options),
			})
		}
		return response, nil
	}
	response.Items = make([]types.QuestionBankAttemptResultItem, 0, len(items))
	for _, item := range items {
		var correctOptions []int
		_ = json.Unmarshal(item.Question.CorrectOptions, &correctOptions)
		sourceAccessible := false
		if item.Question.SourceKnowledgeBaseID != "" {
			if kb, accessErr := s.kbService.GetKnowledgeBaseByIDOnly(ctx, item.Question.SourceKnowledgeBaseID); accessErr == nil && kb != nil {
				if access.AuthorizeKBAccess(ctx, s.kbShare, []*types.KnowledgeBase{kb}) == nil {
					sourceAccessible = true
				}
			}
		}
		sourceKBID, sourceTitle := "", ""
		if sourceAccessible {
			sourceKBID, sourceTitle = item.Question.SourceKnowledgeBaseID, item.Question.SourceTitle
		}
		response.Items = append(response.Items, types.QuestionBankAttemptResultItem{
			ID: item.Question.ID, Type: item.Question.Type, Prompt: item.Question.Prompt,
			Options:         decodeStringArray(item.Question.Options),
			SelectedOptions: item.SelectedOptions, TextAnswer: item.TextAnswer,
			CorrectOptions: correctOptions, ReferenceAnswer: item.Question.ReferenceAnswer,
			Explanation: item.Question.Explanation, Score: item.Score,
			Correct: item.Correct, Feedback: item.Feedback,
			SourceKnowledgeBaseID: sourceKBID, SourceTitle: sourceTitle,
			SourceAccessible: sourceAccessible,
		})
	}
	return response, nil
}

func (s *questionBankService) getBank(ctx context.Context, bankID string) (*types.QuestionBank, error) {
	bank, err := s.repo.GetBank(ctx, types.MustTenantIDFromContext(ctx), bankID)
	if err != nil {
		return nil, err
	}
	if bank == nil {
		return nil, errors.New("question bank not found")
	}
	return bank, nil
}

func (s *questionBankService) defaultModel(ctx context.Context) (chat.Chat, string, error) {
	models, err := s.models.ListModels(ctx)
	if err != nil {
		return nil, "", err
	}
	for _, model := range models {
		if model != nil && model.Type == types.ModelTypeKnowledgeQA && model.Status == types.ModelStatusActive && model.IsDefault {
			chatModel, modelErr := s.models.GetChatModel(ctx, model.ID)
			if modelErr != nil {
				return nil, "", modelErr
			}
			return chatModel, model.ID, nil
		}
	}
	return nil, "", errors.New("no active default chat model is configured for this workspace")
}

func (s *questionBankService) failGeneration(ctx context.Context, bank *types.QuestionBank, cause error) error {
	bank.Status = types.QuestionBankStatusFailed
	bank.GenerationError = cause.Error()
	return s.repo.UpdateBank(ctx, bank)
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validDifficulty(value string) bool {
	switch value {
	case "", "easy", "medium", "hard", "mixed":
		return true
	default:
		return false
	}
}

func decodeStringArray(raw types.JSON) []string {
	var values []string
	if len(raw) == 0 {
		return []string{}
	}
	_ = json.Unmarshal(raw, &values)
	if values == nil {
		return []string{}
	}
	return values
}

func decodeIntArray(raw types.JSON) []int {
	var values []int
	_ = json.Unmarshal(raw, &values)
	return values
}

func decodeAttemptItems(raw types.JSON) ([]types.QuestionBankAttemptItem, error) {
	var items []types.QuestionBankAttemptItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func validateQuestionUpdate(questionType string, req types.QuestionBankQuestionUpdateRequest) error {
	if strings.TrimSpace(req.Prompt) == "" || strings.TrimSpace(req.ReferenceAnswer) == "" {
		return errors.New("question and reference answer are required")
	}
	if questionType == types.QuestionBankQuestionShortAnswer {
		return nil
	}
	if len(req.Options) < 2 || len(req.Options) > 6 {
		return errors.New("choice questions must have between 2 and 6 options")
	}
	if questionType == types.QuestionBankQuestionSingleChoice && len(req.CorrectOptions) != 1 {
		return errors.New("single-choice questions must have exactly one correct option")
	}
	if questionType == types.QuestionBankQuestionMultipleChoice && len(req.CorrectOptions) < 2 {
		return errors.New("multiple-choice questions must have at least two correct options")
	}
	seen := map[int]struct{}{}
	for _, option := range req.CorrectOptions {
		if option < 0 || option >= len(req.Options) {
			return errors.New("correct option index is out of range")
		}
		if _, ok := seen[option]; ok {
			return errors.New("correct options must not contain duplicates")
		}
		seen[option] = struct{}{}
	}
	return nil
}

func applyAnswers(items []types.QuestionBankAttemptItem, answers []types.QuestionBankAnswer) error {
	if len(answers) != len(items) {
		return errors.New("submit one answer for every question")
	}
	byID := make(map[string]types.QuestionBankAnswer, len(answers))
	for _, answer := range answers {
		if answer.QuestionID == "" {
			return errors.New("question_id is required for each answer")
		}
		if _, exists := byID[answer.QuestionID]; exists {
			return errors.New("duplicate answer for question")
		}
		byID[answer.QuestionID] = answer
	}
	for i := range items {
		answer, ok := byID[items[i].Question.ID]
		if !ok {
			return errors.New("answer does not match this attempt")
		}
		items[i].Score = 0
		items[i].Correct = nil
		items[i].Feedback = ""
		items[i].SelectedOptions = nil
		items[i].TextAnswer = ""
		if items[i].Question.Type == types.QuestionBankQuestionShortAnswer {
			if strings.TrimSpace(answer.Text) == "" || len(answer.SelectedOptions) > 0 {
				return errors.New("short-answer questions require text")
			}
			textAnswer := strings.TrimSpace(answer.Text)
			if len([]rune(textAnswer)) > 5000 {
				return errors.New("short-answer responses must be 5000 characters or fewer")
			}
			items[i].TextAnswer = textAnswer
		} else {
			if strings.TrimSpace(answer.Text) != "" {
				return errors.New("choice questions accept selected options only")
			}
			selected := append([]int(nil), answer.SelectedOptions...)
			if err := validateSelectedOptions(items[i].Question, selected); err != nil {
				return err
			}
			items[i].SelectedOptions = selected
		}
		delete(byID, answer.QuestionID)
	}
	if len(byID) != 0 {
		return errors.New("answer contains a question outside this attempt")
	}
	return nil
}

func validateSelectedOptions(question types.QuestionBankQuestion, selected []int) error {
	options := decodeStringArray(question.Options)
	if len(selected) == 0 {
		return errors.New("select at least one option")
	}
	if question.Type == types.QuestionBankQuestionSingleChoice && len(selected) != 1 {
		return errors.New("select exactly one option")
	}
	seen := map[int]struct{}{}
	for _, index := range selected {
		if index < 0 || index >= len(options) {
			return errors.New("selected option index is out of range")
		}
		if _, ok := seen[index]; ok {
			return errors.New("selected options must not contain duplicates")
		}
		seen[index] = struct{}{}
	}
	return nil
}

func gradeObjectiveQuestions(items []types.QuestionBankAttemptItem) error {
	for i := range items {
		if items[i].Question.Type == types.QuestionBankQuestionShortAnswer {
			continue
		}
		correct := sameIntSet(items[i].SelectedOptions, decodeIntArray(items[i].Question.CorrectOptions))
		items[i].Correct = &correct
		if correct {
			items[i].Score = 100
		}
	}
	return nil
}

func sameIntSet(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy, rightCopy := append([]int(nil), left...), append([]int(nil), right...)
	sort.Ints(leftCopy)
	sort.Ints(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func countShortAnswers(items []types.QuestionBankAttemptItem) int {
	count := 0
	for _, item := range items {
		if item.Question.Type == types.QuestionBankQuestionShortAnswer {
			count++
		}
	}
	return count
}
