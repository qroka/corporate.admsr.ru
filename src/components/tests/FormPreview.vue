<script setup lang="ts">
import { computed } from 'vue'
import type { FormWithQuestions } from '../../tests/types'

const props = defineProps<{ form: FormWithQuestions }>()

const sorted = computed(() => [...(props.form.questions ?? [])].sort((a, b) => a.order - b.order))
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-start gap-4">
      <img
        v-if="form.coverUrl"
        :src="form.coverUrl"
        alt=""
        class="w-28 h-28 rounded-xl object-cover border border-default"
      />
      <div class="min-w-0">
        <div class="text-xl font-semibold text-highlighted">{{ form.title }}</div>
        <div v-if="form.description" class="text-sm text-muted mt-1 whitespace-pre-wrap">{{ form.description }}</div>
        <div class="flex gap-2 mt-3">
          <UBadge color="primary" variant="soft">{{ form.mode === 'test' ? 'Тест' : 'Опрос' }}</UBadge>
          <UBadge color="neutral" variant="soft">Вопросов: {{ sorted.length }}</UBadge>
          <UBadge v-if="form.settings?.timeLimitSec" color="amber" variant="soft">
            Лимит: {{ form.settings.timeLimitSec }} сек.
          </UBadge>
        </div>
      </div>
    </div>

    <UCard v-for="q in sorted" :key="q.id">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <div class="font-medium text-highlighted">{{ q.order + 1 }}. {{ q.title }}</div>
          <div v-if="q.hint" class="text-sm text-muted mt-1">{{ q.hint }}</div>
        </div>
        <UBadge v-if="q.required" color="red" variant="soft">обязательный</UBadge>
      </div>

      <div class="mt-4">
        <template v-if="q.type === 'short_text'">
          <UInput disabled placeholder="Ответ..." />
        </template>
        <template v-else-if="q.type === 'long_text'">
          <UTextarea disabled :rows="3" placeholder="Ответ..." />
        </template>
        <template v-else-if="q.type === 'rating_1_10'">
          <div class="grid grid-cols-10 gap-1">
            <UButton v-for="n in 10" :key="n" disabled size="xs" color="neutral" variant="outline">{{ n }}</UButton>
          </div>
        </template>
        <template v-else-if="q.type === 'file'">
          <UInput disabled type="file" />
        </template>
        <template v-else>
          <div class="space-y-2">
            <div v-for="opt in q.options" :key="opt.id" class="flex items-center gap-2 text-sm">
              <span class="w-3 h-3 rounded border border-default" />
              <span class="text-default">{{ opt.label }}</span>
            </div>
          </div>
        </template>
      </div>
    </UCard>
  </div>
</template>

