<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useAppToast } from '../composables/useAppToast'
import { currentRole } from '../stores/role'
import { useTestsStore } from '../tests/store'
import FormBuilder from '../components/tests/FormBuilder.vue'
import FormPreview from '../components/tests/FormPreview.vue'
import FormTake from '../components/tests/FormTake.vue'
import FormReport from '../components/tests/FormReport.vue'
import FormsList from '../components/tests/FormsList.vue'

const toast = useAppToast()
const store = useTestsStore()
store.ensureDraft()

const isAdmin = computed(() => currentRole.value === 'admin')
const tab = ref<'list' | 'builder' | 'take' | 'report'>('list')

const activeForm = computed(() => store.form ?? store.draft)

watch(() => store.error, (e) => {
  if (e) toast.error(String(e))
})

async function loadReport() {
  const id = store.activeFormId
  if (!id) throw new Error('Сначала сохраните форму на сервер и опубликуйте.')
  await store.loadReport(id, {})
}

const tabs = [
  { label: 'Список', value: 'list', icon: 'i-lucide-list' },
  ...(isAdmin.value ? [{ label: 'Конструктор', value: 'builder', icon: 'i-lucide-wrench' } as const] : []),
  { label: 'Прохождение', value: 'take', icon: 'i-lucide-play' },
  ...(isAdmin.value ? [{ label: 'Отчёт', value: 'report', icon: 'i-lucide-bar-chart-3' } as const] : []),
]
</script>

<template>
  <div class="space-y-5">
    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <div class="text-2xl font-semibold text-highlighted">Тесты</div>
        <div class="text-sm text-muted mt-1">
          Конструктор форм (опросы/тесты), прохождение и аналитика.
        </div>
      </div>
      <div class="flex items-center gap-2" v-if="isAdmin">
        <UButton
          color="neutral"
          variant="soft"
          icon="i-lucide-file-plus-2"
          @click="store.setDraft((() => { const d = store.draft!; return { ...d, title: 'Новая форма', description: '', questions: [] } })())"
        >
          Очистить черновик
        </UButton>
        <UButton
          v-if="store.activeFormId"
          color="neutral"
          variant="outline"
          icon="i-lucide-refresh-cw"
          :loading="store.loading"
          @click="store.loadForm(store.activeFormId)"
        >
          Обновить с сервера
        </UButton>
      </div>
    </div>

    <UTabs v-model="tab" :items="tabs" class="w-full" />

    <div v-if="tab === 'list'">
      <FormsList
        :is-admin="isAdmin"
        @open="async (id, action) => {
          await store.loadForm(id)
          if (action === 'take') {
            tab = 'take'
          } else if (action === 'edit') {
            store.setDraftFromLoaded()
            tab = 'builder'
          } else {
            tab = 'report'
          }
        }"
      />
    </div>

    <div v-else-if="tab === 'builder'">
      <FormBuilder>
        <template #preview>
          <FormPreview v-if="activeForm" :form="activeForm" />
        </template>
      </FormBuilder>
    </div>

    <div v-else-if="tab === 'take'">
      <UAlert
        v-if="!store.activeFormId"
        color="amber"
        variant="soft"
        title="Форма ещё не опубликована"
        description="Сохраните и опубликуйте форму в конструкторе. Затем можно проходить."
      />
      <FormTake v-else-if="store.form" :form="store.form" />
      <UCard v-else>
        <div class="text-sm text-muted">
          Загрузите форму с сервера (кнопка «Обновить с сервера») после публикации.
        </div>
      </UCard>
    </div>

    <div v-else>
      <div class="flex items-center justify-between gap-3 mb-3">
        <div class="text-sm text-muted">
          Отчёт доступен после публикации и прохождений.
        </div>
        <UButton
          color="primary"
          icon="i-lucide-refresh-cw"
          :loading="store.loading"
          @click="loadReport().catch(e => toast.error(e instanceof Error ? e.message : 'Ошибка'))"
        >
          Загрузить отчёт
        </UButton>
      </div>
      <FormReport :report="store.report" />
    </div>
  </div>
</template>
