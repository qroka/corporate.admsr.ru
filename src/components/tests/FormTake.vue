<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { AnswerValue, FormWithQuestions, SubmitPayload, UUID } from '../../tests/types'
import { getOrCreateSessionId, progressKey } from '../../tests/utils'
import { useTestsStore } from '../../tests/store'

const props = defineProps<{ form: FormWithQuestions }>()
const store = useTestsStore()

const sorted = computed(() => {
  const q = [...(props.form.questions ?? [])].sort((a, b) => a.order - b.order)
  if (props.form.settings?.shuffleQuestions) {
    // deterministic shuffle by session id (stable per attempt)
    const sid = getOrCreateSessionId(props.form.id)
    let seed = 0
    for (let i = 0; i < sid.length; i++) seed = (seed * 31 + sid.charCodeAt(i)) >>> 0
    const arr = q.slice()
    for (let i = arr.length - 1; i > 0; i--) {
      seed = (seed * 1664525 + 1013904223) >>> 0
      const j = seed % (i + 1)
      ;[arr[i], arr[j]] = [arr[j], arr[i]]
    }
    return arr
  }
  return q
})

const sessionId = ref<UUID>(getOrCreateSessionId(props.form.id))
const answers = ref<Record<UUID, AnswerValue>>({})
const currentIndex = ref(0)
const submitted = ref(false)

const timeLeft = ref<number | null>(null)
let timer: any = null

const canPage = computed(() => (props.form.settings?.pagingMode ?? 'all') === 'by_one')
const currentQuestion = computed(() => sorted.value[currentIndex.value] || null)

function loadProgress() {
  const raw = localStorage.getItem(progressKey(props.form.id, sessionId.value))
  if (!raw) return
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object') answers.value = parsed
  } catch {}
}

function saveProgress() {
  localStorage.setItem(progressKey(props.form.id, sessionId.value), JSON.stringify(answers.value))
}

function startTimer() {
  const limit = props.form.settings?.timeLimitSec ?? null
  if (!limit) return

  // persist endAt so reload keeps correct countdown
  const key = `tests:timerEnd:${props.form.id}:${sessionId.value}`
  const saved = localStorage.getItem(key)
  const endAt = saved ? Number(saved) : Date.now() + limit * 1000
  if (!saved) localStorage.setItem(key, String(endAt))

  const tick = () => {
    const left = Math.max(0, Math.ceil((endAt - Date.now()) / 1000))
    timeLeft.value = left
    if (left <= 0) {
      stopTimer()
      void submit()
    }
  }
  tick()
  timer = setInterval(tick, 1000)
}

function stopTimer() {
  if (timer) clearInterval(timer)
  timer = null
}

function setAnswer(qid: UUID, value: AnswerValue) {
  answers.value = { ...answers.value, [qid]: value }
}

function isAnsweredRequiredMissing() {
  for (const q of sorted.value) {
    if (!q.required) continue
    const a = answers.value[q.id]
    if (!a) return true
    if (a.type === 'single' || a.type === 'select') {
      if (!a.optionId) return true
    } else if (a.type === 'multiple') {
      if (!a.optionIds?.length) return true
    } else if (a.type === 'short_text' || a.type === 'long_text') {
      if (!a.text?.trim()) return true
    } else if (a.type === 'rating_1_10') {
      if (!a.value) return true
    } else if (a.type === 'file') {
      if (!a.base64) return true
    }
  }
  return false
}

async function submit() {
  if (submitted.value) return
  if (isAnsweredRequiredMissing()) {
    throw new Error('Заполните все обязательные вопросы.')
  }

  const payload: SubmitPayload = {
    sessionId: sessionId.value,
    respondent: {},
    answers: Object.entries(answers.value).map(([questionId, value]) => ({ questionId, value })),
  }
  await store.submit(props.form.id, payload)
  submitted.value = true
}

async function onFile(qid: UUID, file: File | null) {
  if (!file) return
  const buf = await file.arrayBuffer()
  const base64 = btoa(String.fromCharCode(...new Uint8Array(buf)))
  setAnswer(qid, { type: 'file', fileName: file.name, mimeType: file.type || 'application/octet-stream', size: file.size, base64 })
}

watch(answers, () => saveProgress(), { deep: true })

onMounted(() => {
  loadProgress()
  startTimer()
})
onBeforeUnmount(() => stopTimer())
</script>

<template>
  <div class="space-y-4">
    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <div class="text-lg font-semibold text-highlighted">{{ form.title }}</div>
            <div class="text-sm text-muted">Сессия: {{ sessionId.slice(0, 8) }}…</div>
          </div>
          <div class="flex items-center gap-3">
            <UBadge v-if="timeLeft !== null" color="amber" variant="soft">
              Осталось: {{ timeLeft }}с
            </UBadge>
            <UButton
              color="primary"
              icon="i-lucide-send"
              :loading="store.loading"
              @click="submit()"
            >
              Отправить
            </UButton>
          </div>
        </div>
      </template>

      <div v-if="store.submitResult" class="mb-4">
        <UAlert
          color="primary"
          variant="soft"
          title="Результат"
          :description="store.submitResult.showResult
            ? `Балл: ${store.submitResult.score ?? '—'} / ${store.submitResult.maxScore ?? '—'}`
            : 'Результат будет доступен после проверки.'"
        />
      </div>

      <template v-if="canPage">
        <div v-if="currentQuestion" class="space-y-3">
          <div class="flex items-start justify-between gap-3">
            <div class="font-medium text-highlighted">
              {{ currentQuestion.order + 1 }}. {{ currentQuestion.title }}
            </div>
            <UBadge v-if="currentQuestion.required" color="red" variant="soft">обязательный</UBadge>
          </div>
          <div v-if="currentQuestion.hint" class="text-sm text-muted">{{ currentQuestion.hint }}</div>

          <div class="mt-2">
            <template v-if="currentQuestion.type === 'single_choice' || currentQuestion.type === 'select'">
              <URadioGroup
                :model-value="(answers[currentQuestion.id] as any)?.optionId ?? null"
                :items="(currentQuestion.options ?? []).map(o => ({ label: o.label, value: o.id }))"
                @update:model-value="(v) => setAnswer(currentQuestion.id, { type: currentQuestion.type === 'select' ? 'select' : 'single', optionId: v ?? null })"
              />
            </template>

            <template v-else-if="currentQuestion.type === 'multiple_choice'">
              <UCheckboxGroup
                :model-value="(answers[currentQuestion.id] as any)?.optionIds ?? []"
                :items="(currentQuestion.options ?? []).map(o => ({ label: o.label, value: o.id }))"
                @update:model-value="(v) => setAnswer(currentQuestion.id, { type: 'multiple', optionIds: (v ?? []) as any })"
              />
            </template>

            <template v-else-if="currentQuestion.type === 'short_text'">
              <UInput
                :model-value="(answers[currentQuestion.id] as any)?.text ?? ''"
                @update:model-value="(v) => setAnswer(currentQuestion.id, { type: 'short_text', text: String(v) })"
              />
            </template>

            <template v-else-if="currentQuestion.type === 'long_text'">
              <UTextarea
                :rows="4"
                :model-value="(answers[currentQuestion.id] as any)?.text ?? ''"
                @update:model-value="(v) => setAnswer(currentQuestion.id, { type: 'long_text', text: String(v) })"
              />
            </template>

            <template v-else-if="currentQuestion.type === 'rating_1_10'">
              <div class="grid grid-cols-10 gap-1">
                <UButton
                  v-for="n in 10"
                  :key="n"
                  :color="((answers[currentQuestion.id] as any)?.value ?? null) === n ? 'primary' : 'neutral'"
                  :variant="((answers[currentQuestion.id] as any)?.value ?? null) === n ? 'solid' : 'outline'"
                  size="xs"
                  @click="setAnswer(currentQuestion.id, { type: 'rating_1_10', value: n })"
                >
                  {{ n }}
                </UButton>
              </div>
            </template>

            <template v-else-if="currentQuestion.type === 'file'">
              <UInput type="file" @change="(e:any) => onFile(currentQuestion!.id, e?.target?.files?.[0] ?? null)" />
              <div v-if="(answers[currentQuestion.id] as any)?.fileName" class="text-xs text-muted mt-2">
                Файл: {{ (answers[currentQuestion.id] as any)?.fileName }}
              </div>
            </template>
          </div>

          <div class="flex items-center justify-between pt-3">
            <UButton color="neutral" variant="outline" :disabled="currentIndex === 0" @click="currentIndex--">
              Назад
            </UButton>
            <div class="text-xs text-muted">
              {{ currentIndex + 1 }} / {{ sorted.length }}
            </div>
            <UButton color="neutral" variant="outline" :disabled="currentIndex >= sorted.length - 1" @click="currentIndex++">
              Далее
            </UButton>
          </div>
        </div>
      </template>

      <template v-else>
        <div class="space-y-4">
          <UCard v-for="q in sorted" :key="q.id">
            <div class="flex items-start justify-between gap-3">
              <div class="font-medium text-highlighted">{{ q.order + 1 }}. {{ q.title }}</div>
              <UBadge v-if="q.required" color="red" variant="soft">обязательный</UBadge>
            </div>
            <div v-if="q.hint" class="text-sm text-muted mt-1">{{ q.hint }}</div>

            <div class="mt-3">
              <template v-if="q.type === 'single_choice' || q.type === 'select'">
                <URadioGroup
                  :model-value="(answers[q.id] as any)?.optionId ?? null"
                  :items="(q.options ?? []).map(o => ({ label: o.label, value: o.id }))"
                  @update:model-value="(v) => setAnswer(q.id, { type: q.type === 'select' ? 'select' : 'single', optionId: v ?? null })"
                />
              </template>
              <template v-else-if="q.type === 'multiple_choice'">
                <UCheckboxGroup
                  :model-value="(answers[q.id] as any)?.optionIds ?? []"
                  :items="(q.options ?? []).map(o => ({ label: o.label, value: o.id }))"
                  @update:model-value="(v) => setAnswer(q.id, { type: 'multiple', optionIds: (v ?? []) as any })"
                />
              </template>
              <template v-else-if="q.type === 'short_text'">
                <UInput
                  :model-value="(answers[q.id] as any)?.text ?? ''"
                  @update:model-value="(v) => setAnswer(q.id, { type: 'short_text', text: String(v) })"
                />
              </template>
              <template v-else-if="q.type === 'long_text'">
                <UTextarea
                  :rows="4"
                  :model-value="(answers[q.id] as any)?.text ?? ''"
                  @update:model-value="(v) => setAnswer(q.id, { type: 'long_text', text: String(v) })"
                />
              </template>
              <template v-else-if="q.type === 'rating_1_10'">
                <div class="grid grid-cols-10 gap-1">
                  <UButton
                    v-for="n in 10"
                    :key="n"
                    :color="((answers[q.id] as any)?.value ?? null) === n ? 'primary' : 'neutral'"
                    :variant="((answers[q.id] as any)?.value ?? null) === n ? 'solid' : 'outline'"
                    size="xs"
                    @click="setAnswer(q.id, { type: 'rating_1_10', value: n })"
                  >
                    {{ n }}
                  </UButton>
                </div>
              </template>
              <template v-else-if="q.type === 'file'">
                <UInput type="file" @change="(e:any) => onFile(q.id, e?.target?.files?.[0] ?? null)" />
                <div v-if="(answers[q.id] as any)?.fileName" class="text-xs text-muted mt-2">
                  Файл: {{ (answers[q.id] as any)?.fileName }}
                </div>
              </template>
            </div>
          </UCard>
        </div>
      </template>
    </UCard>
  </div>
</template>

