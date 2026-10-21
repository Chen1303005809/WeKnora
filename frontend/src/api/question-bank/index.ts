import { get, post, patch } from '@/utils/request'

export type QuestionBankStatus = 'generating' | 'draft' | 'failed' | 'published'
export type QuestionType = 'single_choice' | 'multiple_choice' | 'short_answer'

export interface QuestionBank {
  id: string
  name: string
  description?: string
  grading_prompt?: string
  single_choice_count: number
  multiple_choice_count: number
  short_answer_count: number
  difficulty?: 'easy' | 'medium' | 'hard' | 'mixed' | ''
  attempt_question_count: number
  status: QuestionBankStatus
  generation_error?: string
  created_at: string
  updated_at: string
}

export interface QuestionBankQuestion {
  id: string
  bank_id: string
  type: QuestionType
  prompt: string
  options: string[]
  correct_options: number[]
  reference_answer: string
  explanation?: string
  source_knowledge_base_id?: string
  source_knowledge_base_name?: string
  source_knowledge_id?: string
  source_chunk_id?: string
  source_title?: string
  sort_order: number
  options_text?: string
  correct_options_text?: string
}

export interface QuestionBankAnswer {
  question_id: string
  selected_options?: number[]
  text?: string
}

export interface AttemptQuestion {
  id: string
  type: QuestionType
  prompt: string
  options?: string[]
}

export interface AttemptResultItem extends AttemptQuestion {
  options?: string[]
  selected_options?: number[]
  text_answer?: string
  correct_options?: number[]
  reference_answer: string
  explanation?: string
  score: number
  correct?: boolean
  feedback?: string
  source_knowledge_base_id?: string
  source_title?: string
  source_accessible: boolean
}

export interface QuestionBankAttempt {
  id: string
  bank_id: string
  status: 'in_progress' | 'grading' | 'submitted'
  question_count: number
  score: number
  correct_count: number
  started_at: string
  submitted_at?: string
}

export interface QuestionBankAttemptResponse {
  attempt: QuestionBankAttempt
  questions?: AttemptQuestion[]
  items?: AttemptResultItem[]
}

export async function listPublishedQuestionBanks(): Promise<QuestionBank[]> {
  const response = await get<{ success: boolean; data: QuestionBank[] }>('/api/v1/question-banks')
  return response.data || []
}

export async function listManagedQuestionBanks(): Promise<QuestionBank[]> {
  const response = await get<{ success: boolean; data: QuestionBank[] }>('/api/v1/question-banks/manage')
  return response.data || []
}

export async function getQuestionBank(id: string): Promise<{ bank: QuestionBank; questions: QuestionBankQuestion[] }> {
  const response = await get<{ success: boolean; data: { bank: QuestionBank; questions: QuestionBankQuestion[] } }>(`/api/v1/question-banks/${id}`)
  return response.data
}

export async function createQuestionBank(payload: {
  name: string
  description: string
  source_knowledge_base_ids: string[]
  single_choice_count: number
  multiple_choice_count: number
  short_answer_count: number
  difficulty: string
  attempt_question_count: number
  grading_prompt: string
}): Promise<QuestionBank> {
  const response = await post<{ success: boolean; data: QuestionBank }>('/api/v1/question-banks', payload)
  return response.data
}

export async function updateQuestionBank(id: string, payload: {
  name: string
  description: string
  attempt_question_count: number
  grading_prompt: string
}): Promise<QuestionBank> {
  const response = await patch<{ success: boolean; data: QuestionBank }>(`/api/v1/question-banks/${id}`, payload)
  return response.data
}

export async function updateQuestionBankQuestion(bankID: string, questionID: string, payload: {
  prompt: string
  options: string[]
  correct_options: number[]
  reference_answer: string
  explanation: string
}): Promise<QuestionBankQuestion> {
  const response = await patch<{ success: boolean; data: QuestionBankQuestion }>(`/api/v1/question-banks/${bankID}/questions/${questionID}`, payload)
  return response.data
}

export async function publishQuestionBank(id: string): Promise<QuestionBank> {
  const response = await post<{ success: boolean; data: QuestionBank }>(`/api/v1/question-banks/${id}/publish`)
  return response.data
}

export async function startQuestionBankAttempt(id: string): Promise<QuestionBankAttemptResponse> {
  const response = await post<{ success: boolean; data: QuestionBankAttemptResponse }>(`/api/v1/question-banks/${id}/attempts`)
  return response.data
}

export async function submitQuestionBankAttempt(id: string, answers: QuestionBankAnswer[]): Promise<QuestionBankAttemptResponse> {
  const response = await post<{ success: boolean; data: QuestionBankAttemptResponse }>(`/api/v1/question-banks/attempts/${id}/submit`, { answers })
  return response.data
}

export async function listQuestionBankAttempts(id: string): Promise<QuestionBankAttempt[]> {
  const response = await get<{ success: boolean; data: QuestionBankAttempt[] }>(`/api/v1/question-banks/${id}/attempts`)
  return response.data || []
}

export async function getQuestionBankAttempt(id: string): Promise<QuestionBankAttemptResponse> {
  const response = await get<{ success: boolean; data: QuestionBankAttemptResponse }>(`/api/v1/question-banks/attempts/${id}`)
  return response.data
}
