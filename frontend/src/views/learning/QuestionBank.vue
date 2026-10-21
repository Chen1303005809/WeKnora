<template>
  <main class="learning-page">
    <header class="learning-header">
      <div>
        <p class="eyebrow">{{ t('learningQuiz.title') }}</p>
        <h1>{{ t('learningQuiz.title') }}</h1>
        <p class="subtitle">{{ t('learningQuiz.description') }}</p>
      </div>
      <div class="header-actions">
        <button class="button button--quiet" type="button" :disabled="loading" @click="loadBanks">
          {{ t('learningQuiz.refresh') }}
        </button>
        <button v-if="isAdmin" class="button button--primary" type="button" @click="openCreate">
          <span aria-hidden="true">＋</span>{{ t('learningQuiz.create') }}
        </button>
      </div>
    </header>

    <div v-if="loading && banks.length === 0" class="state-card">{{ t('learningQuiz.loading') }}</div>
    <div v-else-if="banks.length === 0" class="empty-card">
      <div class="empty-icon" aria-hidden="true">✓</div>
      <h2>{{ t('learningQuiz.empty') }}</h2>
      <p>{{ t('learningQuiz.emptyDescription') }}</p>
    </div>
    <section v-else class="bank-grid" aria-label="Question banks">
      <article v-for="bank in banks" :key="bank.id" class="bank-card">
        <div class="bank-card__top">
          <div class="bank-card__title">
            <h2>{{ bank.name }}</h2>
            <span class="status-pill" :class="`status-pill--${bank.status}`">{{ statusLabel(bank.status) }}</span>
          </div>
          <p v-if="bank.description" class="bank-description">{{ bank.description }}</p>
        </div>
        <div class="bank-stats">
          <span>{{ t('learningQuiz.questions', { count: questionCount(bank) }) }}</span>
          <span>{{ t('learningQuiz.attemptCount', { count: bank.attempt_question_count }) }}</span>
        </div>
        <p v-if="bank.status === 'failed' && bank.generation_error" class="bank-error">
          {{ t('learningQuiz.generationError', { error: bank.generation_error }) }}
        </p>
        <div class="bank-card__actions">
          <template v-if="isAdmin && bank.status === 'draft'">
            <button class="button button--quiet" type="button" @click="reviewBank(bank)">{{ t('learningQuiz.review') }}</button>
          </template>
          <template v-else-if="bank.status === 'published'">
            <button class="button button--primary" type="button" @click="startQuiz(bank)">{{ t('learningQuiz.takeQuiz') }}</button>
            <button class="button button--quiet" type="button" @click="showHistory(bank)">{{ t('learningQuiz.history') }}</button>
          </template>
          <span v-else-if="bank.status === 'generating'" class="generating-note"><span class="spinner" />{{ t('learningQuiz.generating') }}</span>
        </div>
      </article>
    </section>

    <div v-if="createVisible" class="overlay" @click.self="createVisible = false">
      <section class="dialog" role="dialog" aria-modal="true" :aria-label="t('learningQuiz.createTitle')">
        <header class="dialog__header">
          <div><p class="eyebrow">{{ t('learningQuiz.create') }}</p><h2>{{ t('learningQuiz.createTitle') }}</h2></div>
          <button class="icon-button" type="button" :aria-label="t('learningQuiz.cancel')" @click="createVisible = false">×</button>
        </header>
        <div class="dialog__body">
          <label class="field">
            <span>{{ t('learningQuiz.name') }}</span>
            <input v-model.trim="createForm.name" maxlength="255" :placeholder="t('learningQuiz.namePlaceholder')" />
          </label>
          <label class="field">
            <span>{{ t('learningQuiz.descriptionLabel') }}</span>
            <textarea v-model="createForm.description" rows="2" />
          </label>
          <fieldset class="field fieldset">
            <legend>{{ t('learningQuiz.knowledgeBases') }}</legend>
            <div v-if="knowledgeBasesLoading" class="hint">{{ t('learningQuiz.loading') }}</div>
            <div v-else-if="knowledgeBases.length === 0" class="hint">{{ t('learningQuiz.noKnowledgeBases') }}</div>
            <label v-for="kb in knowledgeBases" :key="kb.id" class="checkbox-row">
              <input v-model="createForm.source_knowledge_base_ids" type="checkbox" :value="kb.id" />
              <span>{{ kb.name }}</span>
            </label>
          </fieldset>
          <fieldset class="field fieldset">
            <legend>{{ t('learningQuiz.questionCounts') }}</legend>
            <div class="count-grid">
              <label class="field field--compact"><span>{{ t('learningQuiz.singleChoice') }}</span><input v-model.number="createForm.single_choice_count" type="number" min="0" max="50" /></label>
              <label class="field field--compact"><span>{{ t('learningQuiz.multipleChoice') }}</span><input v-model.number="createForm.multiple_choice_count" type="number" min="0" max="50" /></label>
              <label class="field field--compact"><span>{{ t('learningQuiz.shortAnswer') }}</span><input v-model.number="createForm.short_answer_count" type="number" min="0" max="50" /></label>
            </div>
          </fieldset>
          <div class="form-row">
            <label class="field"><span>{{ t('learningQuiz.difficulty') }}</span>
              <select v-model="createForm.difficulty">
                <option value="mixed">{{ t('learningQuiz.mixed') }}</option><option value="easy">{{ t('learningQuiz.easy') }}</option>
                <option value="medium">{{ t('learningQuiz.medium') }}</option><option value="hard">{{ t('learningQuiz.hard') }}</option>
              </select>
            </label>
            <label class="field"><span>{{ t('learningQuiz.perAttempt') }}</span><input v-model.number="createForm.attempt_question_count" type="number" min="1" max="50" /></label>
          </div>
          <label class="field">
            <span>{{ t('learningQuiz.gradingPrompt') }}</span>
            <textarea v-model="createForm.grading_prompt" rows="3" />
          </label>
        </div>
        <footer class="dialog__footer">
          <span v-if="createError" class="inline-error">{{ createError }}</span>
          <button class="button button--quiet" type="button" @click="createVisible = false">{{ t('learningQuiz.cancel') }}</button>
          <button class="button button--primary" type="button" :disabled="creating" @click="createBank">
            {{ creating ? t('learningQuiz.loading') : t('learningQuiz.createAndGenerate') }}
          </button>
        </footer>
      </section>
    </div>

    <div v-if="reviewVisible" class="overlay" @click.self="reviewVisible = false">
      <section class="dialog dialog--wide" role="dialog" aria-modal="true" :aria-label="t('learningQuiz.review')">
        <header class="dialog__header">
          <div><p class="eyebrow">{{ t('learningQuiz.draft') }}</p><h2>{{ reviewForm.name || t('learningQuiz.review') }}</h2></div>
          <button class="icon-button" type="button" :aria-label="t('learningQuiz.cancel')" @click="reviewVisible = false">×</button>
        </header>
        <div class="dialog__body">
          <label class="field"><span>{{ t('learningQuiz.name') }}</span><input v-model.trim="reviewForm.name" maxlength="255" /></label>
          <label class="field"><span>{{ t('learningQuiz.descriptionLabel') }}</span><textarea v-model="reviewForm.description" rows="2" /></label>
          <label class="field"><span>{{ t('learningQuiz.perAttempt') }}</span><input v-model.number="reviewForm.attempt_question_count" type="number" min="1" max="50" /></label>
          <label class="field"><span>{{ t('learningQuiz.gradingPrompt') }}</span><textarea v-model="reviewForm.grading_prompt" rows="3" /></label>
          <p v-if="reviewQuestions.length === 0" class="hint">{{ t('learningQuiz.noQuestions') }}</p>
          <article v-for="(question, index) in reviewQuestions" :key="question.id" class="question-editor">
            <div class="question-editor__heading"><span class="question-number">{{ index + 1 }}</span><strong>{{ questionTypeLabel(question.type) }}</strong><span v-if="question.source_title" class="source-label">{{ question.source_knowledge_base_name }} · {{ question.source_title }}</span></div>
            <label class="field"><span>{{ t('learningQuiz.questionPromptLabel') }}</span><textarea v-model="question.prompt" rows="2" /></label>
            <template v-if="question.type !== 'short_answer'">
              <label class="field"><span>{{ t('learningQuiz.optionsLabel') }}</span><textarea v-model="question.options_text" rows="4" /></label>
              <label class="field"><span>{{ t('learningQuiz.correctOptionsLabel') }}</span><input v-model="question.correct_options_text" :placeholder="'0, 2'" /></label>
            </template>
            <label class="field"><span>{{ t('learningQuiz.referenceAnswer') }}</span><textarea v-model="question.reference_answer" rows="3" /></label>
            <label class="field"><span>{{ t('learningQuiz.explanation') }}</span><textarea v-model="question.explanation" rows="2" /></label>
          </article>
        </div>
        <footer class="dialog__footer dialog__footer--spread">
          <span v-if="reviewError" class="inline-error">{{ reviewError }}</span>
          <div class="footer-actions">
            <button class="button button--quiet" type="button" @click="reviewVisible = false">{{ t('learningQuiz.cancel') }}</button>
            <button class="button button--quiet" type="button" :disabled="savingReview" @click="saveReview">{{ t('learningQuiz.save') }}</button>
            <button class="button button--primary" type="button" :disabled="savingReview || reviewQuestions.length === 0" @click="saveAndPublish">{{ t('learningQuiz.publish') }}</button>
          </div>
        </footer>
      </section>
    </div>

    <div v-if="attemptVisible" class="overlay" @click.self="closeAttempt">
      <section class="dialog dialog--wide" role="dialog" aria-modal="true" :aria-label="attemptResult ? t('learningQuiz.result') : t('learningQuiz.title')">
        <header class="dialog__header">
          <div>
            <p class="eyebrow">{{ attemptResult ? t('learningQuiz.result') : activeBank?.name }}</p>
            <h2 v-if="attemptResult">{{ t('learningQuiz.score', { score: attemptResult.attempt.score.toFixed(2) }) }}</h2>
            <h2 v-else>{{ activeBank?.name }}</h2>
            <p v-if="attemptResult" class="subtitle">{{ t('learningQuiz.correctCount', { correct: attemptResult.attempt.correct_count, count: attemptResult.attempt.question_count }) }}</p>
          </div>
          <button class="icon-button" type="button" :aria-label="t('learningQuiz.cancel')" @click="closeAttempt">×</button>
        </header>
        <div class="dialog__body">
          <template v-if="attemptResult">
            <article v-for="(item, index) in attemptResult.items || []" :key="item.id" class="result-question">
              <div class="question-editor__heading"><span class="question-number">{{ index + 1 }}</span><strong>{{ questionTypeLabel(item.type) }}</strong><b class="result-score">{{ item.score }}%</b></div>
              <h3>{{ item.prompt }}</h3>
              <div v-if="item.options?.length" class="result-options">
                <p v-for="(option, optionIndex) in item.options" :key="optionIndex" :class="{ 'is-correct': item.correct_options?.includes(optionIndex), 'is-selected': item.selected_options?.includes(optionIndex) }">
                  <span>{{ item.selected_options?.includes(optionIndex) ? '●' : '○' }}</span>{{ option }}
                </p>
              </div>
              <p v-if="item.text_answer" class="answer-box">{{ item.text_answer }}</p>
              <p><strong>{{ t('learningQuiz.referenceAnswer') }}:</strong> {{ item.reference_answer }}</p>
              <p v-if="item.explanation"><strong>{{ t('learningQuiz.explanation') }}:</strong> {{ item.explanation }}</p>
              <p v-if="item.feedback"><strong>{{ t('learningQuiz.feedback') }}:</strong> {{ item.feedback }}</p>
              <a v-if="item.source_accessible && item.source_knowledge_base_id" class="source-link" :href="`/platform/knowledge-bases/${item.source_knowledge_base_id}`" @click.prevent="openSource(item.source_knowledge_base_id)">{{ t('learningQuiz.sourceLink') }} · {{ item.source_title }}</a>
            </article>
          </template>
          <template v-else>
            <article v-for="(question, index) in attemptQuestions" :key="question.id" class="quiz-question">
              <div class="question-editor__heading"><span class="question-number">{{ index + 1 }}</span><strong>{{ questionTypeLabel(question.type) }}</strong></div>
              <h3>{{ question.prompt }}</h3>
              <p v-if="question.type === 'multiple_choice'" class="hint">{{ t('learningQuiz.selectOptions') }}</p>
              <label v-for="(option, optionIndex) in question.options || []" :key="optionIndex" class="answer-option">
                <input v-if="question.type === 'single_choice'" type="radio" :name="question.id" :value="optionIndex" :checked="answerMap[question.id].selected_options.includes(optionIndex)" @change="answerMap[question.id].selected_options = [optionIndex]" />
                <input v-else v-model="answerMap[question.id].selected_options" type="checkbox" :value="optionIndex" />
                <span>{{ option }}</span>
              </label>
              <label v-if="question.type === 'short_answer'" class="field">
                <textarea v-model="answerMap[question.id].text" rows="4" :placeholder="t('learningQuiz.answerPlaceholder')" />
              </label>
            </article>
          </template>
        </div>
        <footer v-if="!attemptResult" class="dialog__footer">
          <span v-if="attemptError" class="inline-error">{{ attemptError }}</span>
          <button class="button button--quiet" type="button" @click="closeAttempt">{{ t('learningQuiz.cancel') }}</button>
          <button class="button button--primary" type="button" :disabled="submitting" @click="submitAttempt">
            {{ submitting ? t('learningQuiz.submitting') : t('learningQuiz.submit') }}
          </button>
        </footer>
      </section>
    </div>

    <div v-if="historyVisible" class="overlay" @click.self="historyVisible = false">
      <section class="dialog" role="dialog" aria-modal="true" :aria-label="t('learningQuiz.attemptsTitle')">
        <header class="dialog__header"><div><p class="eyebrow">{{ activeBank?.name }}</p><h2>{{ t('learningQuiz.attemptsTitle') }}</h2></div><button class="icon-button" type="button" @click="historyVisible = false">×</button></header>
        <div class="dialog__body">
          <p v-if="attemptHistory.length === 0" class="hint">{{ t('learningQuiz.noAttempts') }}</p>
          <div v-for="attempt in attemptHistory" :key="attempt.id" class="history-row">
            <div><strong>{{ attempt.status === 'submitted' ? t('learningQuiz.score', { score: attempt.score.toFixed(2) }) : t('learningQuiz.generating') }}</strong><small>{{ t('learningQuiz.startedAt', { time: formatDate(attempt.started_at) }) }}</small></div>
            <button v-if="attempt.status === 'submitted'" class="button button--quiet" type="button" @click="openAttemptResult(attempt.id)">{{ t('learningQuiz.attemptResult') }}</button>
          </div>
        </div>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationStore } from '@/stores/organization'
import {
  createQuestionBank,
  getQuestionBank,
  getQuestionBankAttempt,
  listManagedQuestionBanks,
  listPublishedQuestionBanks,
  listQuestionBankAttempts,
  publishQuestionBank,
  startQuestionBankAttempt,
  submitQuestionBankAttempt,
  updateQuestionBank,
  updateQuestionBankQuestion,
  type AttemptQuestion,
  type QuestionBank,
  type QuestionBankAnswer,
  type QuestionBankAttempt,
  type QuestionBankAttemptResponse,
  type QuestionBankQuestion,
} from '@/api/question-bank'

interface KnowledgeBaseOption { id: string; name: string }
type EditableQuestion = QuestionBankQuestion & { options_text: string; correct_options_text: string }

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const organizationStore = useOrganizationStore()
const isAdmin = computed(() => authStore.hasRole('admin'))
const banks = ref<QuestionBank[]>([])
const knowledgeBases = ref<KnowledgeBaseOption[]>([])
const knowledgeBasesLoading = ref(false)
const loading = ref(false)
const createVisible = ref(false)
const creating = ref(false)
const createError = ref('')
const reviewVisible = ref(false)
const savingReview = ref(false)
const reviewError = ref('')
const reviewBankID = ref('')
const reviewQuestions = ref<EditableQuestion[]>([])
const reviewForm = reactive({ name: '', description: '', attempt_question_count: 10, grading_prompt: '' })
const attemptVisible = ref(false)
const submitting = ref(false)
const attemptError = ref('')
const activeBank = ref<QuestionBank | null>(null)
const attemptID = ref('')
const attemptQuestions = ref<AttemptQuestion[]>([])
const answerMap = reactive<Record<string, { selected_options: number[]; text: string }>>({})
const attemptResult = ref<QuestionBankAttemptResponse | null>(null)
const historyVisible = ref(false)
const attemptHistory = ref<QuestionBankAttempt[]>([])
let pollTimer: ReturnType<typeof setInterval> | undefined

const defaultGradingPrompt = '根据题目、参考答案和评分标准，公平评估学习者的回答。接受语义正确的同义表达；不要求逐字一致。指出回答正确之处和缺少的关键信息。'
const createForm = reactive({
  name: '', description: '', source_knowledge_base_ids: [] as string[],
  single_choice_count: 5, multiple_choice_count: 3, short_answer_count: 2,
  difficulty: 'mixed', attempt_question_count: 10, grading_prompt: defaultGradingPrompt,
})

function questionCount(bank: QuestionBank) {
  return (bank.single_choice_count || 0) + (bank.multiple_choice_count || 0) + (bank.short_answer_count || 0)
}

function statusLabel(status: QuestionBank['status']) {
  const key = status === 'published' ? 'published' : status === 'draft' ? 'draft' : status === 'failed' ? 'failed' : 'generating'
  return t(`learningQuiz.${key}`)
}

function questionTypeLabel(type: string) {
  return type === 'single_choice' ? t('learningQuiz.singleChoice') : type === 'multiple_choice' ? t('learningQuiz.multipleChoice') : t('learningQuiz.shortAnswer')
}

async function loadBanks() {
  if (loading.value) return
  loading.value = true
  try {
    banks.value = isAdmin.value ? await listManagedQuestionBanks() : await listPublishedQuestionBanks()
  } catch {
    MessagePlugin.error(t('learningQuiz.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadKnowledgeBases() {
  knowledgeBasesLoading.value = true
  try {
    const [response] = await Promise.all([
      listKnowledgeBases(),
      organizationStore.fetchSharedKnowledgeBases(),
    ])
    const data: any = Array.isArray(response) ? response : response?.data
    const options = new Map<string, KnowledgeBaseOption>()
    if (Array.isArray(data)) {
      for (const kb of data) {
        if (kb?.id) options.set(kb.id, { id: kb.id, name: kb.name })
      }
    }
    for (const shared of organizationStore.sharedKnowledgeBases) {
      const kb = shared.knowledge_base
      if (kb?.id && !options.has(kb.id)) options.set(kb.id, { id: kb.id, name: kb.name })
    }
    knowledgeBases.value = [...options.values()]
  } catch {
    knowledgeBases.value = []
  } finally {
    knowledgeBasesLoading.value = false
  }
}

function openCreate() {
  createError.value = ''
  createVisible.value = true
  void loadKnowledgeBases()
}

async function createBank() {
  createError.value = ''
  if (!createForm.name.trim() || createForm.source_knowledge_base_ids.length === 0) {
    createError.value = t('learningQuiz.required')
    return
  }
  creating.value = true
  try {
    await createQuestionBank({ ...createForm, name: createForm.name.trim() })
    createVisible.value = false
    await loadBanks()
  } catch {
    createError.value = t('learningQuiz.createFailed')
  } finally {
    creating.value = false
  }
}

async function reviewBank(bank: QuestionBank) {
  reviewError.value = ''
  reviewBankID.value = bank.id
  try {
    const detail = await getQuestionBank(bank.id)
    reviewForm.name = detail.bank.name
    reviewForm.description = detail.bank.description || ''
    reviewForm.attempt_question_count = detail.bank.attempt_question_count || 10
    reviewForm.grading_prompt = detail.bank.grading_prompt || defaultGradingPrompt
    reviewQuestions.value = detail.questions.map(question => ({
      ...question,
      options_text: (question.options || []).join('\n'),
      correct_options_text: (question.correct_options || []).join(', '),
    }))
    reviewVisible.value = true
  } catch {
    MessagePlugin.error(t('learningQuiz.loadFailed'))
  }
}

function serializeQuestion(question: EditableQuestion) {
  const options = question.type === 'short_answer' ? [] : question.options_text.split('\n').map(value => value.trim()).filter(Boolean)
  const correctOptions = question.type === 'short_answer' ? [] : question.correct_options_text.split(',').map(value => Number(value.trim())).filter(value => Number.isInteger(value))
  return {
    prompt: question.prompt,
    options,
    correct_options: correctOptions,
    reference_answer: question.reference_answer,
    explanation: question.explanation || '',
  }
}

async function saveReview() {
  savingReview.value = true
  reviewError.value = ''
  try {
    await updateQuestionBank(reviewBankID.value, {
      name: reviewForm.name,
      description: reviewForm.description,
      attempt_question_count: reviewForm.attempt_question_count,
      grading_prompt: reviewForm.grading_prompt,
    })
    for (const question of reviewQuestions.value) {
      await updateQuestionBankQuestion(reviewBankID.value, question.id, serializeQuestion(question))
    }
    await loadBanks()
    MessagePlugin.success(t('learningQuiz.saveSuccess'))
  } catch (error: any) {
    reviewError.value = error?.message || t('learningQuiz.saveFailed')
  } finally {
    savingReview.value = false
  }
}

async function saveAndPublish() {
  if (!window.confirm(t('learningQuiz.publishConfirm'))) return
  savingReview.value = true
  reviewError.value = ''
  try {
    await updateQuestionBank(reviewBankID.value, {
      name: reviewForm.name,
      description: reviewForm.description,
      attempt_question_count: reviewForm.attempt_question_count,
      grading_prompt: reviewForm.grading_prompt,
    })
    for (const question of reviewQuestions.value) {
      await updateQuestionBankQuestion(reviewBankID.value, question.id, serializeQuestion(question))
    }
    await publishQuestionBank(reviewBankID.value)
    reviewVisible.value = false
    await loadBanks()
  } catch (error: any) {
    reviewError.value = error?.message || t('learningQuiz.saveFailed')
  } finally {
    savingReview.value = false
  }
}

async function startQuiz(bank: QuestionBank) {
  activeBank.value = bank
  attemptError.value = ''
  attemptResult.value = null
  try {
    const response = await startQuestionBankAttempt(bank.id)
    attemptID.value = response.attempt.id
    attemptQuestions.value = response.questions || []
    Object.keys(answerMap).forEach(key => delete answerMap[key])
    for (const question of attemptQuestions.value) answerMap[question.id] = { selected_options: [], text: '' }
    attemptVisible.value = true
  } catch (error: any) {
    attemptError.value = error?.message || t('learningQuiz.submitFailed')
    MessagePlugin.error(attemptError.value)
  }
}

function validateAnswers(): boolean {
  for (const question of attemptQuestions.value) {
    const answer = answerMap[question.id]
    if (question.type === 'short_answer' ? !answer.text.trim() : answer.selected_options.length === 0) {
      attemptError.value = t('learningQuiz.answerRequired')
      return false
    }
  }
  attemptError.value = ''
  return true
}

async function submitAttempt() {
  if (!validateAnswers()) return
  submitting.value = true
  attemptError.value = ''
  try {
    const answers: QuestionBankAnswer[] = attemptQuestions.value.map(question => {
      const answer = answerMap[question.id]
      return question.type === 'short_answer'
        ? { question_id: question.id, text: answer.text.trim() }
        : { question_id: question.id, selected_options: answer.selected_options }
    })
    attemptResult.value = await submitQuestionBankAttempt(attemptID.value, answers)
  } catch (error: any) {
    attemptError.value = error?.message || t('learningQuiz.submitFailed')
  } finally {
    submitting.value = false
  }
}

async function showHistory(bank: QuestionBank) {
  activeBank.value = bank
  historyVisible.value = true
  attemptHistory.value = []
  try {
    attemptHistory.value = await listQuestionBankAttempts(bank.id)
  } catch {
    MessagePlugin.error(t('learningQuiz.loadFailed'))
  }
}

async function openAttemptResult(id: string) {
  try {
    attemptResult.value = await getQuestionBankAttempt(id)
    attemptVisible.value = true
    historyVisible.value = false
  } catch {
    MessagePlugin.error(t('learningQuiz.loadFailed'))
  }
}

function openSource(id: string) {
  void router.push(`/platform/knowledge-bases/${id}`)
  closeAttempt()
}

function closeAttempt() {
  attemptVisible.value = false
  attemptResult.value = null
  attemptQuestions.value = []
}

function formatDate(value: string) {
  return value ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : ''
}

onMounted(() => {
  void loadBanks()
  pollTimer = setInterval(() => {
    if (banks.value.some(bank => bank.status === 'generating')) void loadBanks()
  }, 2500)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped lang="less">
.learning-page { max-width: 1180px; margin: 0 auto; padding: 38px 42px 64px; color: var(--td-text-color-primary, #202a33); }
.learning-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; margin-bottom: 32px; }
.eyebrow { margin: 0 0 6px; color: #07a84f; font-size: 12px; font-weight: 700; letter-spacing: .06em; text-transform: uppercase; }
h1 { margin: 0; font-size: 28px; line-height: 1.25; font-weight: 700; }
.subtitle { margin: 9px 0 0; color: var(--td-text-color-secondary, #697681); font-size: 14px; line-height: 1.6; }
.header-actions, .bank-card__actions, .footer-actions { display: flex; align-items: center; gap: 10px; }
.button { border: 1px solid transparent; border-radius: 8px; padding: 9px 14px; font: inherit; font-size: 13px; font-weight: 600; cursor: pointer; transition: background .15s, border-color .15s, transform .15s; }
.button:hover:not(:disabled) { transform: translateY(-1px); }
.button:disabled { opacity: .55; cursor: not-allowed; }
.button--primary { background: #07c05f; color: white; box-shadow: 0 3px 8px #07c05f2c; }
.button--primary:hover:not(:disabled) { background: #05a951; }
.button--quiet { color: #2b3640; background: var(--td-bg-color-container, #fff); border-color: var(--td-component-border, #dfe5e9); }
.button--quiet:hover:not(:disabled) { background: var(--td-bg-color-container-hover, #f5f8f7); border-color: #b9d8c6; }
.bank-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(290px, 1fr)); gap: 18px; }
.bank-card, .empty-card, .state-card { background: var(--td-bg-color-container, #fff); border: 1px solid var(--td-component-border, #e5eaed); border-radius: 14px; box-shadow: 0 5px 18px #1b28330a; }
.bank-card { display: flex; min-height: 215px; flex-direction: column; padding: 21px; }
.bank-card__top { flex: 1; }
.bank-card__title { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.bank-card h2 { margin: 0; font-size: 17px; line-height: 1.4; overflow-wrap: anywhere; }
.status-pill { flex: none; border-radius: 999px; padding: 4px 9px; color: #54616a; background: #f0f3f4; font-size: 11px; font-weight: 650; }
.status-pill--published { color: #087a3d; background: #e7f8ee; }.status-pill--draft { color: #795500; background: #fff5d6; }.status-pill--failed { color: #af3333; background: #fff0ed; }.status-pill--generating { color: #286eaa; background: #eaf4ff; }
.bank-description { margin: 12px 0; color: var(--td-text-color-secondary, #697681); font-size: 13px; line-height: 1.6; }
.bank-stats { display: flex; flex-wrap: wrap; gap: 7px 14px; margin-top: 18px; color: #65717a; font-size: 12px; }
.bank-error { margin: 12px 0 0; color: #bb4545; font-size: 12px; line-height: 1.45; }
.bank-card__actions { margin-top: 19px; }
.empty-card, .state-card { max-width: 680px; margin: 72px auto; padding: 42px 24px; text-align: center; }
.empty-card h2 { margin: 18px 0 7px; font-size: 19px; }.empty-card p { margin: 0; color: #737d84; font-size: 14px; }
.empty-icon { display: grid; width: 56px; height: 56px; margin: auto; place-items: center; border-radius: 16px; color: #07954b; background: #eaf9f0; font-size: 27px; }
.generating-note { display: inline-flex; align-items: center; gap: 8px; color: #4c718e; font-size: 13px; }
.spinner { width: 14px; height: 14px; border: 2px solid #bdd5e5; border-top-color: #1684c4; border-radius: 50%; animation: spin .8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.overlay { position: fixed; z-index: 3000; inset: 0; display: flex; align-items: center; justify-content: center; padding: 24px; background: #16222d80; backdrop-filter: blur(3px); }
.dialog { display: flex; width: min(620px, 100%); max-height: min(88vh, 920px); flex-direction: column; overflow: hidden; border: 1px solid #e3e9ec; border-radius: 16px; background: var(--td-bg-color-container, #fff); box-shadow: 0 22px 70px #111a2440; }
.dialog--wide { width: min(820px, 100%); }
.dialog__header { display: flex; flex: none; align-items: flex-start; justify-content: space-between; gap: 20px; padding: 23px 25px 17px; border-bottom: 1px solid #edf0f2; }
.dialog__header h2 { margin: 0; font-size: 20px; line-height: 1.35; }.dialog__header .subtitle { margin-top: 4px; }
.icon-button { width: 34px; height: 34px; border: 0; border-radius: 8px; color: #606c75; background: transparent; font-size: 25px; cursor: pointer; }.icon-button:hover { background: #f1f4f5; }
.dialog__body { padding: 20px 25px 24px; overflow: auto; }
.dialog__footer { display: flex; flex: none; align-items: center; justify-content: flex-end; gap: 10px; padding: 15px 25px; border-top: 1px solid #edf0f2; }.dialog__footer--spread { justify-content: space-between; }.footer-actions { margin-left: auto; }
.field { display: flex; flex-direction: column; gap: 7px; margin: 0 0 17px; color: #39444c; font-size: 12px; font-weight: 650; }
.field input, .field textarea, .field select { box-sizing: border-box; width: 100%; border: 1px solid #dce3e7; border-radius: 8px; padding: 10px 11px; color: #24303a; background: var(--td-bg-color-container, #fff); font: inherit; font-size: 13px; font-weight: 400; outline: none; }
.field textarea { resize: vertical; line-height: 1.55; }.field input:focus, .field textarea:focus, .field select:focus { border-color: #48bf7c; box-shadow: 0 0 0 3px #07c05f18; }
.fieldset { min-width: 0; border: 0; border-top: 1px solid #edf0f2; padding: 15px 0 0; }.fieldset legend { padding: 0 8px 0 0; font-size: 12px; font-weight: 700; }
.checkbox-row { display: flex; align-items: center; gap: 9px; min-height: 32px; color: #46525b; font-size: 13px; font-weight: 400; }.checkbox-row input { accent-color: #07b759; }
.count-grid, .form-row { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }.form-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
.field--compact { margin-bottom: 0; }.inline-error { margin-right: auto; color: #bd3f3f; font-size: 12px; line-height: 1.4; }.hint { color: #7a858d; font-size: 12px; line-height: 1.5; }
.question-editor, .result-question, .quiz-question { margin: 0 0 18px; padding: 18px; border: 1px solid #e8edef; border-radius: 12px; background: #fcfdfd; }.question-editor:last-child, .result-question:last-child, .quiz-question:last-child { margin-bottom: 0; }
.question-editor__heading { display: flex; align-items: center; gap: 9px; margin-bottom: 13px; color: #49565e; font-size: 12px; }.question-number { display: inline-grid; width: 24px; height: 24px; place-items: center; border-radius: 8px; color: #078846; background: #e7f8ed; font-size: 12px; font-weight: 700; }.source-label { margin-left: auto; overflow: hidden; color: #849098; font-weight: 400; text-overflow: ellipsis; white-space: nowrap; }.question-editor .field:last-child { margin-bottom: 0; }
.source-link { display: inline-block; margin-top: 3px; color: #078b48; font-size: 12px; text-decoration: none; }.source-link:hover { text-decoration: underline; }
.quiz-question h3, .result-question h3 { margin: 0 0 14px; font-size: 15px; line-height: 1.6; font-weight: 600; }.answer-option { display: flex; align-items: flex-start; gap: 10px; padding: 9px 10px; border-radius: 8px; color: #39464e; font-size: 13px; line-height: 1.45; cursor: pointer; }.answer-option:hover { background: #f1f8f4; }.answer-option input { margin-top: 2px; accent-color: #06b858; }
.result-question > p { margin: 8px 0; color: #5a666f; font-size: 13px; line-height: 1.55; }.result-question > p strong { color: #303c44; }.result-score { margin-left: auto; color: #087a3d; }.result-options p { display: flex; gap: 8px; margin: 5px 0; color: #69757e; font-size: 13px; }.result-options .is-correct { color: #087a3d; font-weight: 650; }.result-options .is-selected:not(.is-correct) { color: #bb4242; }.answer-box { border-radius: 7px; padding: 10px; background: #f3f6f7; }
.history-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 0; border-bottom: 1px solid #edf0f2; }.history-row:last-child { border-bottom: 0; }.history-row strong { display: block; font-size: 14px; }.history-row small { display: block; margin-top: 5px; color: #7a858d; font-size: 11px; }
@media (max-width: 700px) { .learning-page { padding: 24px 16px 44px; }.learning-header { flex-direction: column; }.header-actions { width: 100%; }.bank-grid { grid-template-columns: 1fr; }.overlay { align-items: flex-end; padding: 8px; }.dialog { max-height: 93vh; border-radius: 15px 15px 10px 10px; }.dialog__body { padding: 17px; }.dialog__header, .dialog__footer { padding-left: 17px; padding-right: 17px; }.count-grid { grid-template-columns: 1fr; gap: 10px; }.form-row { grid-template-columns: 1fr; gap: 0; }.dialog__footer--spread { align-items: flex-end; flex-direction: column; }.footer-actions { width: 100%; justify-content: flex-end; }.inline-error { align-self: flex-start; } }
</style>
