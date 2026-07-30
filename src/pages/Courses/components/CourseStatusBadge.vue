<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  status?: string | null;
}>();

const meta = computed(() => {
  const s = String(props.status || '').toLowerCase().trim();
  const map: Record<string, { label: string; color: 'neutral' | 'primary' | 'success' | 'warning' | 'error' | 'info' }> = {
    // курс / версия
    draft: { label: 'Черновик', color: 'neutral' },
    published: { label: 'Опубликован', color: 'success' },
    archived: { label: 'В архиве', color: 'warning' },
    // назначение
    not_started: { label: 'Не начат', color: 'neutral' },
    in_progress: { label: 'В процессе', color: 'primary' },
    completed: { label: 'Завершён', color: 'success' },
    overdue: { label: 'Просрочен', color: 'error' },
    failed: { label: 'Не сдан', color: 'error' },
    cancelled: { label: 'Отменён', color: 'neutral' },
    // тема
    locked: { label: 'Закрыта', color: 'neutral' },
    available: { label: 'Доступна', color: 'info' },
    // материал
    opened: { label: 'Открыт', color: 'primary' },
  };
  return map[s] || { label: 'Статус неизвестен', color: 'neutral' as const };
});
</script>

<template>
  <UBadge :color="meta.color" variant="subtle" :label="meta.label" />
</template>
