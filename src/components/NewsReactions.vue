<script setup lang="ts">
/**
 * Реакции на новость: чипы с количеством (моя реакция подсвечена), «+» — выбор
 * из всех реакций. Подсказка над чипом показывает, кто отреагировал.
 * readonly — киоск и гости: только просмотр.
 */
import { computed, ref } from 'vue';
import {
  NEWS_REACTIONS,
  useNewsReactions,
  type NewsReactionKey,
} from '../composables/useNewsReactions';

const props = withDefaults(
  defineProps<{
    newsId: string | number;
    readonly?: boolean;
    size?: 'xs' | 'sm';
  }>(),
  { readonly: false, size: 'xs' },
);

const { reactionsOf, countOf, isMine, toggle, loadReactors, reactorsOf } = useNewsReactions();

const pickerOpen = ref(false);

/** «Нравится» видно всегда; остальные — только если кто-то их поставил. */
const chips = computed(() => {
  const list = reactionsOf(props.newsId);
  return NEWS_REACTIONS.filter((r) => r.key === 'like' || list.some((x) => x.key === r.key));
});

function formatCount(n: number) {
  return n > 0 ? n.toLocaleString('ru-RU') : '';
}

function tooltipText(key: NewsReactionKey, label: string): string {
  const count = countOf(props.newsId, key);
  if (!count) return props.readonly ? label : `${label} — будьте первым`;
  const people = reactorsOf(props.newsId, key);
  if (!people) return `${label}: ${count}`;
  const names = people.slice(0, 5).map((p) => p.name);
  const rest = count - names.length;
  return `${label}: ${names.join(', ')}${rest > 0 ? ` и ещё ${rest}` : ''}`;
}

function onTooltip(open: boolean, key: NewsReactionKey) {
  if (open && !props.readonly && countOf(props.newsId, key) > 0) void loadReactors(props.newsId, key);
}

function onPick(key: NewsReactionKey) {
  pickerOpen.value = false;
  void toggle(props.newsId, key);
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-1.5" @click.stop>
    <UTooltip
      v-for="r in chips"
      :key="r.key"
      :text="tooltipText(r.key, r.label)"
      @update:open="onTooltip($event, r.key)"
    >
      <UButton
        type="button"
        :size="size"
        :color="isMine(newsId, r.key) ? 'primary' : 'neutral'"
        variant="subtle"
        :square="!countOf(newsId, r.key)"
        :disabled="readonly"
        :aria-label="`${r.label}: ${countOf(newsId, r.key)}`"
        :aria-pressed="isMine(newsId, r.key)"
        class="tabular-nums justify-center disabled:opacity-100"
        @click="toggle(newsId, r.key)"
      >
        <!-- Без числа кнопка квадратная и эмодзи по центру, с числом — «эмодзи + число». -->
        <span class="text-sm leading-4" aria-hidden="true">{{ r.emoji }}</span>
        <span v-if="countOf(newsId, r.key)">{{ formatCount(countOf(newsId, r.key)) }}</span>
      </UButton>
    </UTooltip>

    <UPopover v-if="!readonly" v-model:open="pickerOpen" :content="{ side: 'top', align: 'start', sideOffset: 6 }">
      <UTooltip text="Добавить реакцию">
        <UButton
          type="button"
          :size="size"
          color="neutral"
          variant="ghost"
          square
          icon="i-lucide-smile-plus"
          aria-label="Добавить реакцию"
        />
      </UTooltip>

      <template #content>
        <div class="grid grid-cols-4 gap-1 p-1.5" role="group" aria-label="Выберите реакцию">
          <UTooltip v-for="r in NEWS_REACTIONS" :key="r.key" :text="r.label">
            <UButton
              type="button"
              size="md"
              :color="isMine(newsId, r.key) ? 'primary' : 'neutral'"
              :variant="isMine(newsId, r.key) ? 'subtle' : 'ghost'"
              square
              :aria-label="r.label"
              :aria-pressed="isMine(newsId, r.key)"
              @click="onPick(r.key)"
            >
              <span class="text-xl leading-none" aria-hidden="true">{{ r.emoji }}</span>
            </UButton>
          </UTooltip>
        </div>
      </template>
    </UPopover>
  </div>
</template>
