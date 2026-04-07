<script setup lang="ts">
import { computed, ref } from 'vue'
import draggable from 'vuedraggable'
import { useTestsStore } from '../../tests/store'
import { FormWithQuestionsUpsertSchema, validatePublishable } from '../../tests/schemas'
import QuestionEditor from './QuestionEditor.vue'

const store = useTestsStore()
store.ensureDraft()

const showPreview = ref(false)

const statusBadge = computed(() => {
  const s = store.form?.status ?? store.draft?.status ?? 'draft'
  if (s === 'published') return { label: 'Опубликован', color: 'green' as const }
  if (s === 'archived') return { label: 'Архив', color: 'neutral' as const }
  return { label: 'Черновик', color: 'amber' as const }
})

const modeItems = [
  { label: 'Опрос', value: 'survey' },
  { label: 'Тест', value: 'test' },
]

function updateDraft(patch: Partial<any>) {
  const d = store.draft!
  store.setDraft({ ...d, ...patch })
}

function updateQuestion(id: string, next: any) {
  store.updateQuestion(id, () => next)
}

async function onSave() {
  // client-side shape validation (server will revalidate)
  const parsed = FormWithQuestionsUpsertSchema.safeParse({ form: store.draft, questions: store.draft?.questions ?? [] })
  if (!parsed.success) throw new Error('Проверьте поля формы (ошибка валидации).')
  await store.saveDraftToServer()
}

async function onUpdate() {
  const parsed = FormWithQuestionsUpsertSchema.safeParse({ form: store.draft, questions: store.draft?.questions ?? [] })
  if (!parsed.success) throw new Error('Проверьте поля формы (ошибка валидации).')
  await store.updateActiveOnServer()
}

async function onPublish() {
  const d = store.draft
  if (!d) return
  const chk = validatePublishable({ form: d as any, questions: d.questions as any })
  if (!chk.ok) throw new Error(chk.error)

  if (!store.activeFormId) {
    await store.saveDraftToServer()
  } else {
    await store.updateActiveOnServer()
  }
  await store.publishActive()
}
</script>

<template>
  <div class="space-y-5">
    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <div class="text-lg font-semibold text-highlighted">Конструктор</div>
            <UBadge :color="statusBadge.color" variant="soft">{{ statusBadge.label }}</UBadge>
          </div>
          <div class="flex items-center gap-2">
            <UButton color="neutral" variant="soft" icon="i-lucide-eye" @click="showPreview = true">
              Предпросмотр
            </UButton>
            <UButton
              color="neutral"
              variant="outline"
              icon="i-lucide-save"
              :loading="store.loading"
              @click="store.activeFormId ? onUpdate() : onSave()"
            >
              {{ store.activeFormId ? 'Обновить' : 'Сохранить' }}
            </UButton>
            <UButton
              color="primary"
              icon="i-lucide-upload"
              :loading="store.loading"
              @click="onPublish()"
            >
              Опубликовать
            </UButton>
          </div>
        </div>
      </template>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <UFormField label="Заголовок">
          <UInput :model-value="store.draft?.title" @update:model-value="(v) => updateDraft({ title: String(v) })" />
        </UFormField>
        <UFormField label="Тип">
          <USelect
            :items="modeItems"
            :model-value="store.draft?.mode"
            @update:model-value="(v) => updateDraft({ mode: v })"
          />
        </UFormField>
        <UFormField label="Обложка (URL)">
          <UInput
            :model-value="store.draft?.coverUrl ?? ''"
            placeholder="https://..."
            @update:model-value="(v) => updateDraft({ coverUrl: String(v || '').trim() || null })"
          />
        </UFormField>
      </div>

      <div class="mt-4">
        <UFormField label="Описание">
          <UTextarea
            :model-value="store.draft?.description"
            :rows="3"
            @update:model-value="(v) => updateDraft({ description: String(v) })"
          />
        </UFormField>
      </div>

      <div class="mt-6 grid grid-cols-1 lg:grid-cols-4 gap-4">
        <UFormField label="Лимит времени (сек.)">
          <UInput
            type="number"
            :model-value="store.draft?.settings?.timeLimitSec ?? ''"
            placeholder="—"
            @update:model-value="(v) => updateDraft({ settings: { ...(store.draft?.settings ?? {}), timeLimitSec: v === '' ? null : Number(v) } })"
          />
        </UFormField>
        <UFormField label="Перемешивать вопросы">
          <UToggle
            :model-value="Boolean(store.draft?.settings?.shuffleQuestions)"
            @update:model-value="(v) => updateDraft({ settings: { ...(store.draft?.settings ?? {}), shuffleQuestions: Boolean(v) } })"
          />
        </UFormField>
        <UFormField label="Показывать результат">
          <USelect
            :items="[
              { label: 'Сразу', value: 'immediate' },
              { label: 'После проверки', value: 'after_review' },
            ]"
            :model-value="store.draft?.settings?.showResultMode ?? 'immediate'"
            @update:model-value="(v) => updateDraft({ settings: { ...(store.draft?.settings ?? {}), showResultMode: v } })"
          />
        </UFormField>
        <UFormField label="Попыток">
          <UInput
            type="number"
            :model-value="store.draft?.settings?.attemptsLimit ?? ''"
            placeholder="—"
            @update:model-value="(v) => updateDraft({ settings: { ...(store.draft?.settings ?? {}), attemptsLimit: v === '' ? null : Number(v) } })"
          />
        </UFormField>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div class="text-base font-semibold text-highlighted">Вопросы</div>
          <div class="flex items-center gap-2">
            <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="store.addQuestion()">
              Добавить вопрос
            </UButton>
          </div>
        </div>
      </template>

      <div v-if="store.questions.length === 0" class="text-sm text-muted">
        Добавьте хотя бы один вопрос.
      </div>

      <draggable
        v-else
        :model-value="store.questions"
        item-key="id"
        handle=".drag-handle"
        class="space-y-4"
        @update:model-value="(v) => store.reorderQuestions(v)"
      >
        <template #item="{ element }">
          <div>
            <div class="flex items-center gap-2 mb-2">
              <UButton class="drag-handle" icon="i-lucide-grip-vertical" variant="ghost" color="neutral" size="xs" />
              <span class="text-xs text-muted">Перетащите, чтобы изменить порядок</span>
            </div>
            <QuestionEditor
              :model-value="element"
              :mode="store.draft?.mode ?? 'survey'"
              @update:model-value="(v) => updateQuestion(element.id, v)"
              @delete="store.deleteQuestion(element.id)"
            />
          </div>
        </template>
      </draggable>
    </UCard>

    <UModal v-model:open="showPreview" :ui="{ width: 'max-w-4xl' }">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div class="font-semibold text-highlighted">Предпросмотр</div>
            <UButton color="neutral" variant="ghost" icon="i-lucide-x" @click="showPreview = false" />
          </div>
        </template>
        <slot name="preview" />
      </UCard>
    </UModal>
  </div>
</template>

