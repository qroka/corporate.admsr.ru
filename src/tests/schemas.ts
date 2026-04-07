import { z } from 'zod'

const uuid = z.string().uuid()

export const FormSettingsSchema = z.object({
  timeLimitSec: z.number().int().min(10).max(24 * 3600).nullable().optional(),
  shuffleQuestions: z.boolean().optional(),
  showResultMode: z.enum(['immediate', 'after_review']).optional(),
  attemptsLimit: z.number().int().min(1).max(100).nullable().optional(),
  pagingMode: z.enum(['all', 'by_one']).optional(),
})

export const FormUpsertSchema = z.object({
  title: z.string().trim().min(1).max(200),
  description: z.string().trim().max(2000).default(''),
  coverUrl: z.string().trim().url().nullable().optional(),
  status: z.enum(['draft', 'published', 'archived']).optional(),
  mode: z.enum(['test', 'survey']).default('survey'),
  settings: FormSettingsSchema.default({}),
})

export const QuestionOptionSchema = z.object({
  id: uuid.optional(),
  label: z.string().trim().min(1).max(300),
  isCorrect: z.boolean().optional(),
  order: z.number().int().min(0),
})

export const QuestionSchema = z.object({
  id: uuid.optional(),
  type: z.enum([
    'single_choice',
    'multiple_choice',
    'short_text',
    'long_text',
    'rating_1_10',
    'select',
    'file',
  ]),
  order: z.number().int().min(0),
  title: z.string().trim().min(1).max(500),
  hint: z.string().trim().max(1000).default(''),
  required: z.boolean().default(false),
  options: z.array(QuestionOptionSchema).optional(),
})

export const FormWithQuestionsUpsertSchema = z.object({
  form: FormUpsertSchema,
  questions: z.array(QuestionSchema).default([]),
})

export function validatePublishable(payload: z.infer<typeof FormWithQuestionsUpsertSchema>) {
  const q = payload.questions ?? []
  if (q.length < 1) return { ok: false as const, error: 'Перед публикацией добавьте хотя бы 1 вопрос.' }

  for (const question of q) {
    const hasOptions = question.type === 'single_choice' || question.type === 'multiple_choice' || question.type === 'select'
    if (hasOptions) {
      const opts = question.options ?? []
      if (opts.length < 2) {
        return { ok: false as const, error: 'Вопросы с выбором должны иметь минимум 2 варианта.' }
      }

      if (payload.form.mode === 'test') {
        const anyCorrect = opts.some(o => o.isCorrect === true)
        if (!anyCorrect) {
          return { ok: false as const, error: 'В тесте у вопроса должен быть отмечен хотя бы один правильный вариант.' }
        }
      }
    }
  }

  return { ok: true as const }
}
