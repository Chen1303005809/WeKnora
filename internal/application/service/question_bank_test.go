package service

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestQuestionBankObjectiveScoringRequiresExactOptionSet(t *testing.T) {
	items := []types.QuestionBankAttemptItem{
		{
			Question:        types.QuestionBankQuestion{ID: "single", Type: types.QuestionBankQuestionSingleChoice, Options: types.JSON(`["A","B"]`), CorrectOptions: types.JSON(`[1]`)},
			SelectedOptions: []int{1},
		},
		{
			Question:        types.QuestionBankQuestion{ID: "multiple", Type: types.QuestionBankQuestionMultipleChoice, Options: types.JSON(`["A","B","C"]`), CorrectOptions: types.JSON(`[0,2]`)},
			SelectedOptions: []int{0, 1, 2},
		},
	}
	if err := gradeObjectiveQuestions(items); err != nil {
		t.Fatal(err)
	}
	if items[0].Score != 100 || items[0].Correct == nil || !*items[0].Correct {
		t.Fatalf("single-choice score = %#v", items[0])
	}
	if items[1].Score != 0 || items[1].Correct == nil || *items[1].Correct {
		t.Fatalf("multiple-choice score = %#v", items[1])
	}
}

func TestApplyQuestionBankAnswersRejectsQuestionsOutsideAttempt(t *testing.T) {
	items := []types.QuestionBankAttemptItem{{
		Question: types.QuestionBankQuestion{ID: "expected", Type: types.QuestionBankQuestionShortAnswer},
	}}
	err := applyAnswers(items, []types.QuestionBankAnswer{{QuestionID: "other", Text: "answer"}})
	if err == nil {
		t.Fatal("expected a mismatched answer to be rejected")
	}
}

func TestQuestionBankModelResponseMayBeWrappedInMarkdownFence(t *testing.T) {
	var response generatedQuestionResponse
	err := unmarshalQuestionBankJSON("```json\n{\"questions\":[]}\n```", &response)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Questions) != 0 {
		t.Fatalf("questions = %#v", response.Questions)
	}
}
