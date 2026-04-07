<script setup lang="ts">
import { computed, h, onMounted, ref, resolveComponent } from 'vue'
import type { TableColumn } from '@nuxt/ui'
import type { FormListItem, UUID } from '../../tests/types'
import { useTestsStore } from '../../tests/store'

const props = defineProps<{
  isAdmin: boolean
}>()

const emit = defineEmits<{
  (e: 'open', id: UUID, action: 'take' | 'edit' | 'report'): void
}>()

const store = useTestsStore()

const UBadge = resolveComponent('UBadge')
const UButton = resolveComponent('UButton')

const q = ref('')
const status = ref<'all' | 'draft' | 'published' | 'archived'>('all')

const statusItems = computed(() => {
  if (!props.isAdmin) return [{ label: 'Опубликованные', value: 'published' }]
  return [
    { label: 'Все', value: 'all' },
    { label: 'Черновики', value: 'draft' },
    { label: 'Опубликованные', value: 'published' },
    { label: 'Архив', value: 'archived' },
  ]
})

const rows = computed<FormListItem[]>(() => store.formsList)

const columns = computed<TableColumn<FormListItem>[]>(() => {
  const base: TableColumn<FormListItem>[] = [
    {
      accessorKey: 'title',
      header: 'Название',
      cell: ({ row }) => {
        const r = row.original as FormListItem
        return h('div', { class: 'min-w-0' }, [
          h('div', { class: 'font-medium text-highlighted truncate' }, r.title),
          r.description
            ? h('div', { class: 'text-xs text-muted truncate' }, r.description)
            : null,
        ])
      },
    },
    {
      accessorKey: 'mode',
      header: 'Тип',
      cell: ({ row }) => {
        const mode = String(row.getValue('mode') ?? '')
        return h(UBadge, { color: mode === 'test' ? 'primary' : 'neutral', variant: 'soft' }, () =>
          mode === 'test' ? 'Тест' : 'Опрос',
        )
      },
    },
    {
      accessorKey: 'status',
      header: 'Статус',
      cell: ({ row }) => {
        const s = String(row.getValue('status') ?? '')
        const color = s === 'published' ? 'green' : s === 'archived' ? 'neutral' : 'amber'
        const label = s === 'published' ? 'Опубликован' : s === 'archived' ? 'Архив' : 'Черновик'
        return h(UBadge, { color, variant: 'soft' }, () => label)
      },
    },
    { accessorKey: 'questionsCount', header: 'Вопросов' },
    {
      accessorKey: 'createdAt',
      header: 'Создано',
      cell: ({ row }) => {
        const dt = new Date(String(row.getValue('createdAt') ?? ''))
        return h('div', { class: 'text-sm text-muted' }, Number.isNaN(dt.getTime()) ? '—' : dt.toLocaleString('ru-RU'))
      },
    },
    {
      id: 'actions',
      header: '',
      enableHiding: false,
      meta: { class: { th: 'w-[1%] whitespace-nowrap', td: 'text-right whitespace-nowrap' } },
      cell: ({ row }) => {
        const r = row.original as FormListItem
        const kids: any[] = []

        if (r.status === 'published') {
          kids.push(
            h(UButton, { color: 'primary', variant: 'soft', size: 'sm', icon: 'i-lucide-play', onClick: () => emit('open', r.id, 'take') }, () => 'Пройти'),
          )
        }

        if (props.isAdmin) {
          kids.push(
            h(UButton, { color: 'neutral', variant: 'outline', size: 'sm', icon: 'i-lucide-pen-line', onClick: () => emit('open', r.id, 'edit') }, () => 'Редакт.'),
          )
          kids.push(
            h(UButton, { color: 'neutral', variant: 'outline', size: 'sm', icon: 'i-lucide-bar-chart-3', onClick: () => emit('open', r.id, 'report') }, () => 'Отчёт'),
          )
          if (r.status !== 'archived') {
            kids.push(
              h(UButton, { color: 'neutral', variant: 'soft', size: 'sm', icon: 'i-lucide-archive', onClick: () => openConfirm('archive', r.id) }, () => 'В архив'),
            )
          } else {
            kids.push(
              h(UButton, { color: 'neutral', variant: 'soft', size: 'sm', icon: 'i-lucide-archive-restore', onClick: () => openConfirm('unarchive', r.id) }, () => 'Вернуть'),
            )
          }
          kids.push(
            h(UButton, { color: 'red', variant: 'soft', size: 'sm', icon: 'i-lucide-trash-2', onClick: () => openConfirm('delete', r.id) }, () => 'Удалить'),
          )
        }

        return h('div', { class: 'flex items-center justify-end gap-2' }, kids)
      },
    },
  ]
  return base
})

const confirmOpen = ref(false)
const confirmKind = ref<'archive' | 'unarchive' | 'delete'>('archive')
const confirmFormId = ref<UUID | null>(null)

function openConfirm(kind: 'archive' | 'unarchive' | 'delete', id: UUID) {
  confirmKind.value = kind
  confirmFormId.value = id
  confirmOpen.value = true
}

async function runConfirm() {
  const id = confirmFormId.value
  if (!id) return
  if (confirmKind.value === 'delete') {
    await store.deleteForm(id)
  } else if (confirmKind.value === 'archive') {
    await store.archiveForm(id, 'archived')
  } else {
    await store.archiveForm(id, 'draft')
  }
  confirmOpen.value = false
  await reload()
}

async function reload() {
  const filters: any = {}
  filters.q = q.value.trim() || undefined
  if (props.isAdmin) {
    filters.status = status.value === 'all' ? undefined : status.value
  } else {
    filters.status = 'published'
  }
  await store.loadFormsList(filters)
}

onMounted(() => {
  if (!props.isAdmin) status.value = 'published'
  void reload()
})
</script>

<template>
  <UCard>
    <template #header>
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <div class="text-lg font-semibold text-highlighted">Список форм</div>
          <div class="text-sm text-muted">
            {{ isAdmin ? 'Управление черновиками, публикациями и архивом.' : 'Доступны только опубликованные формы.' }}
          </div>
        </div>
        <div class="flex items-center gap-2">
          <UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="store.loading" @click="reload">
            Обновить
          </UButton>
        </div>
      </div>
    </template>

    <div class="flex flex-col md:flex-row gap-3 mb-4">
      <UInput v-model="q" placeholder="Поиск по названию..." class="flex-1" @keyup.enter="reload" />
      <USelect v-model="status" :items="statusItems" class="w-full md:w-60" :disabled="!isAdmin" />
      <UButton color="primary" icon="i-lucide-search" :loading="store.loading" @click="reload">Найти</UButton>
    </div>

    <UTable
      :data="rows"
      :columns="columns"
    />
  </UCard>

  <UModal v-model:open="confirmOpen" :ui="{ width: 'max-w-lg' }">
    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="font-semibold text-highlighted">Подтверждение</div>
          <UButton color="neutral" variant="ghost" icon="i-lucide-x" @click="confirmOpen = false" />
        </div>
      </template>

      <div class="text-sm text-default">
        <template v-if="confirmKind === 'delete'">
          Удалить форму без возможности восстановления?
        </template>
        <template v-else-if="confirmKind === 'archive'">
          Переместить форму в архив?
        </template>
        <template v-else>
          Вернуть форму из архива (в статус черновика)?
        </template>
      </div>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="outline" @click="confirmOpen = false">Отмена</UButton>
          <UButton
            :color="confirmKind === 'delete' ? 'red' : 'primary'"
            :loading="store.loading"
            @click="runConfirm()"
          >
            Подтвердить
          </UButton>
        </div>
      </template>
    </UCard>
  </UModal>
</template>

