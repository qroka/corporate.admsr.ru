export type UUID = string

export type FormStatus = 'draft' | 'published' | 'archived'
export type FormMode = 'test' | 'survey'

export type QuestionType =
  | 'single_choice'
  | 'multiple_choice'
  | 'short_text'
  | 'long_text'
  | 'rating_1_10'
  | 'select'
  | 'file'

export type FormSettings = {
  timeLimitSec?: number | null
  shuffleQuestions?: boolean
  showResultMode?: 'immediate' | 'after_review'
  attemptsLimit?: number | null
  pagingMode?: 'all' | 'by_one'
}

export type Form = {
  id: UUID
  title: string
  description: string
  coverUrl: string | null
  status: FormStatus
  mode: FormMode
  settings: FormSettings
  createdAt: string
  updatedAt: string
}

export type FormListItem = Pick<Form, 'id' | 'title' | 'description' | 'coverUrl' | 'status' | 'mode' | 'createdAt' | 'updatedAt'> & {
  questionsCount: number
}

export type QuestionOption = {
  id: UUID
  label: string
  isCorrect?: boolean
  order: number
}

export type Question = {
  id: UUID
  formId: UUID
  type: QuestionType
  order: number
  title: string
  hint: string
  required: boolean
  options?: QuestionOption[]
  correct?: {
    optionId?: UUID | null
    optionIds?: UUID[]
  }
}

export type FormWithQuestions = Form & {
  questions: Question[]
}

export type AnswerValue =
  | { type: 'single'; optionId: UUID | null }
  | { type: 'multiple'; optionIds: UUID[] }
  | { type: 'short_text'; text: string }
  | { type: 'long_text'; text: string }
  | { type: 'rating_1_10'; value: number | null }
  | { type: 'select'; optionId: UUID | null }
  | { type: 'file'; fileName: string; mimeType: string; size: number; base64: string }

export type SubmitPayload = {
  sessionId: UUID
  respondent?: {
    userId?: string | number | null
    fio?: string | null
    email?: string | null
  }
  answers: Array<{
    questionId: UUID
    value: AnswerValue
  }>
}

export type SubmitResult = {
  responseId: UUID
  score: number | null
  maxScore: number | null
  correctPercent: number | null
  showResult: boolean
  perQuestion?: Array<{
    questionId: UUID
    isCorrect: boolean | null
    earned: number | null
    possible: number | null
  }>
}

export type Report = {
  form: Form
  summary: {
    totalResponses: number
    completedResponses: number
    avgScore: number | null
    medianScore: number | null
    stdDevScore: number | null
    correctPercentAvg: number | null
  }
  funnel: Array<{
    questionId: UUID
    reached: number
  }>
  topMistakes: Array<{
    questionId: UUID
    wrongPercent: number
  }>
  questions: Array<{
    questionId: UUID
    type: QuestionType
    title: string
    distribution: Array<{ key: string; label: string; count: number }>
  }>
  participants: Array<{
    responseId: UUID
    sessionId: UUID
    userId: string | null
    fio: string | null
    startedAt: string
    completedAt: string | null
    status: 'completed' | 'incomplete'
    score: number | null
    maxScore: number | null
  }>
}

export type UUID = string

export type FormStatus = 'draft' | 'published' | 'archived'
export type FormMode = 'test' | 'survey'

export type QuestionType =
  | 'single_choice'
  | 'multiple_choice'
  | 'short_text'
  | 'long_text'
  | 'rating_1_10'
  | 'select'
  | 'file'

export type FormSettings = {
  timeLimitSec?: number | null
  shuffleQuestions?: boolean
  showResultMode?: 'immediate' | 'after_review'
  attemptsLimit?: number | null
  pagingMode?: 'all' | 'by_one'
}

export type Form = {
  id: UUID
  title: string
  description: string
  coverUrl: string | null
  status: FormStatus
  mode: FormMode
  settings: FormSettings
  createdAt: string
  updatedAt: string
}

export type QuestionOption = {
  id: UUID
  label: string
  isCorrect?: boolean
  order: number
}

export type Question = {
  id: UUID
  formId: UUID
  type: QuestionType
  order: number
  title: string
  hint: string
  required: boolean
  options?: QuestionOption[]
  // Для вопросов без вариантов: может быть пусто
  correct?: {
    // Для single/select: optionId
    optionId?: UUID | null
    // Для multiple: optionIds
    optionIds?: UUID[]
    // Для rating/text — можно расширять позже
  }
}

export type FormWithQuestions = Form & {
  questions: Question[]
}

export type Session = {
  sessionId: UUID
}

export type AnswerValue =
  | { type: 'single'; optionId: UUID | null }
  | { type: 'multiple'; optionIds: UUID[] }
  | { type: 'short_text'; text: string }
  | { type: 'long_text'; text: string }
  | { type: 'rating_1_10'; value: number | null }
  | { type: 'select'; optionId: UUID | null }
  | { type: 'file'; fileName: string; mimeType: string; size: number; base64: string }

export type SubmitPayload = {
  sessionId: UUID
  respondent?: {
    userId?: string | number | null
    fio?: string | null
    email?: string | null
  }
  answers: Array<{
    questionId: UUID
    value: AnswerValue
  }>
}

export type SubmitResult = {
  responseId: UUID
  score: number | null
  maxScore: number | null
  correctPercent: number | null
  showResult: boolean
  perQuestion?: Array<{
    questionId: UUID
    isCorrect: boolean | null
    earned: number | null
    possible: number | null
  }>
}

export type Report = {
  form: Form
  summary: {
    totalResponses: number
    completedResponses: number
    avgScore: number | null
    medianScore: number | null
    stdDevScore: number | null
    correctPercentAvg: number | null
  }
  funnel: Array<{
    questionId: UUID
    reached: number
  }>
  topMistakes: Array<{
    questionId: UUID
    wrongPercent: number
  }>
  questions: Array<{
    questionId: UUID
    type: QuestionType
    title: string
    distribution: Array<{ key: string; label: string; count: number }>
  }>
  participants: Array<{
    responseId: UUID
    sessionId: UUID
    userId: string | null
    fio: string | null
    startedAt: string
    completedAt: string | null
    status: 'completed' | 'incomplete'
    score: number | null
    maxScore: number | null
  }>
}

