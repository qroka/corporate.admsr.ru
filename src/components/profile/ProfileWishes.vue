<script setup lang="ts">
/**
 * «Желания» — список желаний сотрудника, как «Желания Павла» у старого ВКонтакте.
 * Добавляет и удаляет только владелец; видят все вошедшие. Сервер: profile_extras.php.
 */
import { computed, ref } from 'vue';
import { apiSessionFetch } from '../../composables/useAuthSession';
import { useAppToast } from '../../composables/useAppToast';
import ProfileSection from './ProfileSection.vue';

export type Wish = { id: number; text: string };

const props = defineProps<{ wishes: Wish[]; isOwn: boolean }>();
const emit = defineEmits<{ (e: 'update', wishes: Wish[]): void }>();

const MAX_LENGTH = 200;
const MAX_COUNT = 20;

const { error } = useAppToast();
const draft = ref('');
const adding = ref(false);
const removingId = ref<number | null>(null);

const canAdd = computed(() => {
  const text = draft.value.trim();
  return !!text && text.length <= MAX_LENGTH && props.wishes.length < MAX_COUNT && !adding.value;
});

async function call(body: Record<string, unknown>, failTitle: string): Promise<Wish[] | null> {
  try {
    const res = await apiSessionFetch<{ wishes: Wish[] }>('/api/profile_extras.php', { method: 'POST', json: body });
    if (!res?.success) {
      error(failTitle, res?.message);
      return null;
    }
    return res.data?.wishes ?? [];
  } catch {
    error(failTitle, 'Проверьте подключение к сети и попробуйте ещё раз.');
    return null;
  }
}

async function add() {
  if (!canAdd.value) return;
  adding.value = true;
  const list = await call({ action: 'wish_add', text: draft.value.trim() }, 'Желание не добавлено');
  adding.value = false;
  if (list) {
    draft.value = '';
    emit('update', list);
  }
}

async function remove(wish: Wish) {
  removingId.value = wish.id;
  const list = await call({ action: 'wish_delete', id: wish.id }, 'Желание не удалено');
  removingId.value = null;
  if (list) emit('update', list);
}
</script>

<template>
  <ProfileSection v-if="wishes.length || isOwn" title="Желания" title-id="profile-wishes">
    <template v-if="wishes.length" #aside>{{ wishes.length }}</template>

    <ul v-if="wishes.length" class="flex flex-col gap-1.5">
      <li v-for="w in wishes" :key="w.id" class="flex items-start gap-2 text-sm">
        <UIcon name="i-lucide-sparkles" class="size-4 text-dimmed shrink-0 mt-0.5" aria-hidden="true" />
        <span class="flex-1 min-w-0 text-default break-words">{{ w.text }}</span>
        <UTooltip v-if="isOwn" text="Удалить желание">
          <UButton
            type="button"
            color="neutral"
            variant="ghost"
            size="xs"
            square
            icon="i-lucide-x"
            :loading="removingId === w.id"
            :aria-label="`Удалить желание: ${w.text}`"
            @click="remove(w)"
          />
        </UTooltip>
      </li>
    </ul>
    <p v-else class="text-sm text-muted">Здесь можно записать, чему хотите научиться или что хотите сделать.</p>

    <form v-if="isOwn" class="flex gap-2 pt-3" @submit.prevent="add">
      <UInput
        v-model="draft"
        size="sm"
        class="flex-1 min-w-0"
        :maxlength="MAX_LENGTH"
        :disabled="wishes.length >= MAX_COUNT"
        :placeholder="wishes.length >= MAX_COUNT ? 'Максимум 20 желаний' : 'Новое желание'"
        aria-label="Новое желание"
      />
      <UTooltip text="Добавить желание">
        <UButton type="submit" size="sm" color="neutral" variant="outline" square icon="i-lucide-plus" :loading="adding" :disabled="!canAdd" aria-label="Добавить желание" />
      </UTooltip>
    </form>
  </ProfileSection>
</template>
