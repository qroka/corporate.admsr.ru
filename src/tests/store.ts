import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { FormListItem, FormWithQuestions, Question, UUID, SubmitPayload, SubmitResult, Report } from './types'
import { apiArchiveForm, apiCreateForm, apiDeleteForm, apiGetForm, apiGetReport, apiListForms, apiPublishForm, apiSubmitForm, apiUpdateForm } from './api'

const DRAFT_KEY = 'tests:draft'

function safeJson<T>(raw: string | null): T | null {
  if (!raw) return null
  try {
    return JSON.parse(raw) as T
  } catch {
    return null
  }
}

export const useTestsStore = defineStore('tests', () => {
  const activeFormId = ref<UUID | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const form = ref<FormWithQuestions | null>(null)
  const report = ref<Report | null>(null)
  const submitResult = ref<SubmitResult | null>(null)
  const formsList = ref<FormListItem[]>([])

  const draft = ref<FormWithQuestions | null>(safeJson<FormWithQuestions>(localStorage.getItem(DRAFT_KEY)))

  const questions = computed(() => form.value?.questions ?? draft.value?.questions ?? [])

  function setDraft(next: FormWithQuestions) {
    draft.value = next
    localStorage.setItem(DRAFT_KEY, JSON.stringify(next))
  }

  function newDraft(): FormWithQuestions {
    const now = new Date().toISOString()
    return {
      id: crypto.randomUUID(),
      title: 'Новая форма',
      description: '',
      coverUrl: null,
      status: 'draft',
      mode: 'survey',
      settings: { pagingMode: 'all', showResultMode: 'immediate', shuffleQuestions: false, attemptsLimit: null, timeLimitSec: null },
      createdAt: now,
      updatedAt: now,
      questions: [],
    }
  }

  function ensureDraft() {
    if (!draft.value) setDraft(newDraft())
  }

  function addQuestion(partial?: Partial<Question>) {
    ensureDraft()
    const d = draft.value!
    const order = d.questions.length
    const q: Question = {
      id: crypto.randomUUID(),
      formId: d.id,
      type: partial?.type ?? 'single_choice',
      order,
      title: partial?.title ?? 'Вопрос',
      hint: partial?.hint ?? '',
      required: partial?.required ?? false,
      options: partial?.options ?? (partial?.type && ['short_text', 'long_text', 'rating_1_10', 'file'].includes(partial.type) ? undefined : [
        { id: crypto.randomUUID(), label: 'Вариант 1', order: 0, isCorrect: false },
        { id: crypto.randomUUID(), label: 'Вариант 2', order: 1, isCorrect: false },
      ]),
    }
    setDraft({ ...d, questions: [...d.questions, q] })
  }

  function updateQuestion(id: UUID, updater: (q: Question) => Question) {
    ensureDraft()
    const d = draft.value!
    setDraft({
      ...d,
      questions: d.questions.map(q => (q.id === id ? updater(q) : q)),
    })
  }

  function deleteQuestion(id: UUID) {
    ensureDraft()
    const d = draft.value!
    const next = d.questions.filter(q => q.id !== id).map((q, idx) => ({ ...q, order: idx }))
    setDraft({ ...d, questions: next })
  }

  function reorderQuestions(next: Question[]) {
    ensureDraft()
    const d = draft.value!
    setDraft({ ...d, questions: next.map((q, idx) => ({ ...q, order: idx })) })
  }

  async function loadForm(id: UUID) {
    loading.value = true
    error.value = null
    try {
      const data = await apiGetForm(id)
      form.value = data
      activeFormId.value = id
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка загрузки формы'
    } finally {
      loading.value = false
    }
  }

  function setDraftFromLoaded() {
    if (!form.value) return
    setDraft(form.value)
  }

  async function loadFormsList(filters: { status?: string; q?: string } = {}) {
    loading.value = true
    error.value = null
    try {
      formsList.value = await apiListForms(filters)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка загрузки списка'
      formsList.value = []
    } finally {
      loading.value = false
    }
  }

  async function archiveForm(id: UUID, nextStatus: 'archived' | 'draft') {
    loading.value = true
    error.value = null
    try {
      await apiArchiveForm(id, nextStatus)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка архивирования'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteForm(id: UUID) {
    loading.value = true
    error.value = null
    try {
      await apiDeleteForm(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка удаления'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function saveDraftToServer() {
    ensureDraft()
    const d = draft.value!
    loading.value = true
    error.value = null
    try {
      const created = await apiCreateForm(d)
      await apiUpdateForm(created.id, d)
      await loadForm(created.id)
      // после успешного сохранения можно оставить draft как локальную копию
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка сохранения'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateActiveOnServer() {
    const id = activeFormId.value
    const d = draft.value
    if (!id || !d) return
    loading.value = true
    error.value = null
    try {
      await apiUpdateForm(id, d)
      await loadForm(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка обновления'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function publishActive() {
    const id = activeFormId.value
    if (!id) throw new Error('Нет активной формы')
    loading.value = true
    error.value = null
    try {
      await apiPublishForm(id)
      await loadForm(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка публикации'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function submit(formId: UUID, payload: SubmitPayload) {
    loading.value = true
    error.value = null
    try {
      const r = await apiSubmitForm(formId, payload)
      submitResult.value = r
      return r
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка отправки'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function loadReport(formId: UUID, filters: Record<string, any>) {
    loading.value = true
    error.value = null
    try {
      report.value = await apiGetReport(formId, filters)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Ошибка отчёта'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    activeFormId,
    loading,
    error,
    form,
    draft,
    questions,
    report,
    submitResult,
    formsList,
    ensureDraft,
    setDraft,
    setDraftFromLoaded,
    addQuestion,
    updateQuestion,
    deleteQuestion,
    reorderQuestions,
    loadForm,
    loadFormsList,
    archiveForm,
    deleteForm,
    saveDraftToServer,
    updateActiveOnServer,
    publishActive,
    submit,
    loadReport,
  }
})

