<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { formatRelativeRu } from '../../utils/date';
import { useAppToast } from '../../composables/useAppToast';

const SUBSCRIBE_KEY = 'news-subscriptions:v1';

const props = withDefaults(
  defineProps<{
    id: string;
    title: string;
    description: string;
    imageSrc: string;
    imageAlt?: string;
    to: string;
    date?: string | null;
    createdAt?: string | null;
    views: number;
    authorName?: string;
    authorRole?: string;
    authorAvatar?: string;
  }>(),
  {
    imageAlt: 'Новость',
    authorName: 'Редакция портала',
    authorRole: 'Новости',
  },
);

const { toast } = useAppToast();
const subscribed = ref(false);

function readSubscribed(): Record<string, boolean> {
  if (typeof window === 'undefined') return {};
  try {
    return JSON.parse(window.localStorage.getItem(SUBSCRIBE_KEY) || '{}') ?? {};
  } catch {
    return {};
  }
}

function writeSubscribed(map: Record<string, boolean>) {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(SUBSCRIBE_KEY, JSON.stringify(map));
  } catch {
    /* ignore */
  }
}

watch(
  () => props.id,
  (id) => {
    subscribed.value = !!readSubscribed()[String(id)];
  },
  { immediate: true },
);

const relativeTime = computed(() => {
  const raw = props.createdAt || props.date;
  return formatRelativeRu(raw) || '';
});

const avatarProps = computed(() => {
  if (props.authorAvatar) return { src: props.authorAvatar, alt: props.authorName };
  const initials = props.authorName
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase() ?? '')
    .join('');
  return { text: initials || 'РП', alt: props.authorName };
});

function toggleSubscribe() {
  const key = String(props.id);
  const map = readSubscribed();
  const next = !subscribed.value;
  map[key] = next;
  if (!next) delete map[key];
  writeSubscribed(map);
  subscribed.value = next;
  toast.add({
    title: next ? 'Подписка оформлена' : 'Подписка отменена',
    description: next
      ? 'Уведомления о материале сохранены локально (без бэкенда).'
      : undefined,
    color: 'primary',
    icon: next ? 'i-lucide-bell' : 'i-lucide-bell-off',
  });
}

const lightboxOpen = ref(false);

function downloadImage(e?: Event) {
  e?.preventDefault();
  e?.stopPropagation();
  if (!props.imageSrc) return;
  const a = document.createElement('a');
  a.href = props.imageSrc;
  a.download = `news-${props.id}`;
  a.target = '_blank';
  a.rel = 'noopener';
  a.click();
}

function openLightbox(e?: Event) {
  e?.preventDefault();
  e?.stopPropagation();
  if (!props.imageSrc) return;
  lightboxOpen.value = true;
}
</script>

<template>
  <UCard
    variant="soft"
    class="w-full h-[300px] shrink-0 rounded-panel"
    :ui="{
      root: 'divide-y-0 h-[300px] rounded-panel bg-elevated ring-0 border-0 overflow-hidden',
      body: 'p-0 sm:p-0 h-full overflow-hidden',
    }"
  >
    <div class="grid h-full grid-cols-2">
      <div class="group relative h-full min-w-0 overflow-hidden bg-muted">
        <img
          :src="imageSrc"
          alt=""
          aria-hidden="true"
          class="absolute inset-0 size-full scale-110 object-cover blur-2xl"
          loading="lazy"
          decoding="async"
        />
        <img
          :src="imageSrc"
          :alt="imageAlt"
          class="relative z-10 block size-full object-contain"
          loading="lazy"
          decoding="async"
        />
        <div
          class="pointer-events-none absolute inset-0 z-20 opacity-0 group-hover:opacity-100 transition"
        >
          <div class="absolute top-2 right-2 flex gap-1 pointer-events-auto">
            <UTooltip text="Скачать">
              <UButton
                type="button"
                color="neutral"
                variant="solid"
                size="xs"
                icon="i-lucide-download"
                square
                class="bg-black/60 hover:bg-black/80 text-white ring-0"
                aria-label="Скачать"
                @click="downloadImage($event)"
              />
            </UTooltip>
            <UTooltip text="Открыть">
              <UButton
                type="button"
                color="neutral"
                variant="solid"
                size="xs"
                icon="i-lucide-expand"
                square
                class="bg-black/60 hover:bg-black/80 text-white ring-0"
                aria-label="Открыть"
                @click="openLightbox($event)"
              />
            </UTooltip>
          </div>
        </div>
      </div>

      <div class="flex h-full min-w-0 flex-col gap-2.5 p-4">
        <div class="flex shrink-0 items-center gap-2.5">
          <UUser
            :name="authorName"
            :description="authorRole"
            :avatar="avatarProps"
            size="md"
            class="min-w-0 flex-1"
          />
          <UButton
            type="button"
            size="sm"
            color="neutral"
            variant="solid"
            class="shrink-0 bg-inverted text-inverted hover:bg-inverted/90"
            :icon="subscribed ? 'i-lucide-check' : 'i-lucide-plus'"
            :aria-label="subscribed ? 'Вы подписаны' : 'Подписаться'"
            @click.stop="toggleSubscribe"
          >
            <span class="hidden sm:inline">{{ subscribed ? 'Вы подписаны' : 'Подписаться' }}</span>
          </UButton>
        </div>

        <div class="flex min-h-0 flex-1 flex-col gap-1.5">
          <h3 class="shrink-0 text-lg font-bold leading-6 text-highlighted text-pretty line-clamp-2">
            {{ title }}
          </h3>
          <p class="min-h-0 text-sm font-medium leading-5 text-default text-pretty line-clamp-3">
            {{ description }}
          </p>
          <UButton
            :to="to"
            color="neutral"
            variant="subtle"
            size="sm"
            class="mt-auto self-start shrink-0"
          >
            Подробнее
          </UButton>
          <p v-if="relativeTime" class="shrink-0 text-xs leading-4 text-muted">
            {{ relativeTime }}
          </p>
        </div>

        <USeparator
          class="shrink-0"
          :ui="{ border: 'border-inverted/20' }"
        />

        <NewsReactions :news-id="id" class="shrink-0" />
      </div>
    </div>
  </UCard>

  <UModal
    v-model:open="lightboxOpen"
    class="p-0"
    :ui="{ content: 'bg-transparent shadow-none ring-0 w-auto max-w-[95vw]', header: 'hidden', body: 'p-0' }"
  >
    <template #content>
      <div class="flex flex-col items-center justify-center gap-3 p-0">
        <img
          :src="imageSrc"
          :alt="imageAlt"
          decoding="async"
          class="block w-auto h-auto max-w-[95vw] max-h-[85vh] rounded-lg"
        />
      </div>
    </template>
  </UModal>
</template>
