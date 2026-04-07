<script setup lang="ts">
import { h, onMounted, onBeforeUnmount, ref, resolveComponent, watch } from 'vue'
import type { TableColumn } from '@nuxt/ui'
import type { Report } from '../../tests/types'
import { Chart, BarController, BarElement, CategoryScale, LinearScale, PieController, ArcElement, Tooltip, Legend } from 'chart.js'

Chart.register(BarController, BarElement, CategoryScale, LinearScale, PieController, ArcElement, Tooltip, Legend)

const props = defineProps<{ report: Report | null }>()

const charts = ref<Chart[]>([])

const UBadge = resolveComponent('UBadge')

const participantColumns = ref<TableColumn<any>[]>([
  {
    accessorKey: 'startedAt',
    header: 'Старт',
    cell: ({ row }) => {
      const dt = new Date(String(row.getValue('startedAt') ?? ''))
      return h('span', { class: 'text-sm text-muted' }, Number.isNaN(dt.getTime()) ? '—' : dt.toLocaleString('ru-RU'))
    },
  },
  {
    accessorKey: 'status',
    header: 'Статус',
    cell: ({ row }) => {
      const s = String(row.getValue('status') ?? '')
      return h(UBadge, { color: s === 'completed' ? 'green' : 'amber', variant: 'soft' }, () => (s === 'completed' ? 'Завершён' : 'Не завершён'))
    },
  },
  { accessorKey: 'fio', header: 'ФИО' },
  { accessorKey: 'userId', header: 'UserID' },
  { accessorKey: 'score', header: 'Балл' },
])

function destroyCharts() {
  for (const c of charts.value) c.destroy()
  charts.value = []
}

function build() {
  destroyCharts()
  const r = props.report
  if (!r) return

  for (const q of r.questions) {
    const el = document.getElementById(`chart-${q.questionId}`) as HTMLCanvasElement | null
    if (!el) continue

    const labels = q.distribution.map(d => d.label)
    const values = q.distribution.map(d => d.count)

    const isPie = q.distribution.length <= 6 && (q.type === 'single_choice' || q.type === 'select')
    const chart = new Chart(el, {
      type: isPie ? 'pie' : 'bar',
      data: {
        labels,
        datasets: [
          {
            label: 'Ответы',
            data: values,
            backgroundColor: isPie
              ? ['#3b82f6', '#22c55e', '#f59e0b', '#ef4444', '#a855f7', '#14b8a6']
              : '#3b82f6',
          },
        ],
      },
      options: {
        responsive: true,
        plugins: { legend: { display: isPie } },
        scales: isPie
          ? undefined
          : {
              y: { beginAtZero: true, ticks: { precision: 0 } },
            },
      },
    })
    charts.value.push(chart)
  }
}

onMounted(build)
onBeforeUnmount(destroyCharts)
watch(() => props.report, () => build())
</script>

<template>
  <div v-if="!report" class="text-sm text-muted">Нет данных отчёта.</div>

  <div v-else class="space-y-5">
    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="font-semibold text-highlighted">Сводка</div>
          <div class="flex gap-2">
            <UButton
              color="neutral"
              variant="outline"
              icon="i-lucide-download"
              :to="`/api/forms_report.php?id=${encodeURIComponent(report.form.id)}&format=csv`"
              target="_blank"
            >
              Экспорт CSV
            </UButton>
            <UButton color="neutral" variant="soft" icon="i-lucide-printer" @click="window.print()">
              PDF (печать)
            </UButton>
          </div>
        </div>
      </template>

      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <UCard>
          <div class="text-xs text-muted">Прохождений</div>
          <div class="text-2xl font-semibold text-highlighted">{{ report.summary.totalResponses }}</div>
        </UCard>
        <UCard>
          <div class="text-xs text-muted">Завершили</div>
          <div class="text-2xl font-semibold text-highlighted">{{ report.summary.completedResponses }}</div>
        </UCard>
        <UCard>
          <div class="text-xs text-muted">Средний балл</div>
          <div class="text-2xl font-semibold text-highlighted">{{ report.summary.avgScore ?? '—' }}</div>
        </UCard>
        <UCard>
          <div class="text-xs text-muted">Медиана / σ</div>
          <div class="text-2xl font-semibold text-highlighted">
            {{ report.summary.medianScore ?? '—' }} / {{ report.summary.stdDevScore ?? '—' }}
          </div>
        </UCard>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="font-semibold text-highlighted">Воронка завершения</div>
      </template>
      <div class="space-y-2">
        <div v-for="(f, idx) in report.funnel" :key="f.questionId" class="flex items-center gap-3">
          <div class="w-10 text-xs text-muted">#{{ idx + 1 }}</div>
          <UProgress :value="report.summary.totalResponses ? (f.reached / report.summary.totalResponses) * 100 : 0" />
          <div class="w-20 text-right text-sm text-highlighted">{{ f.reached }}</div>
        </div>
      </div>
    </UCard>

    <UCard v-if="report.topMistakes.length">
      <template #header>
        <div class="font-semibold text-highlighted">Топ-3 вопросов с ошибками</div>
      </template>
      <div class="space-y-2">
        <div v-for="t in report.topMistakes" :key="t.questionId" class="flex items-center justify-between">
          <div class="text-sm text-default">Вопрос {{ t.questionId.slice(0, 8) }}…</div>
          <UBadge color="red" variant="soft">{{ t.wrongPercent.toFixed(1) }}%</UBadge>
        </div>
      </div>
    </UCard>

    <div class="space-y-4">
      <UCard v-for="q in report.questions" :key="q.questionId">
        <template #header>
          <div class="font-semibold text-highlighted">{{ q.title }}</div>
        </template>
        <div class="h-[260px]">
          <canvas :id="`chart-${q.questionId}`" class="w-full h-full" />
        </div>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <div class="font-semibold text-highlighted">Участники</div>
      </template>
      <UTable
        :data="report.participants"
        :columns="participantColumns"
      />
    </UCard>
  </div>
</template>

