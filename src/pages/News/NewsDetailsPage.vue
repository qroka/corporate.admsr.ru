<script setup lang="ts">
import { computed, watch } from 'vue';
import { useRoute } from 'vue-router';
import { useNewsData, formatUnixDate, resolveNewsImageSrc } from '../../composables/useNewsData';
import { useAppToast } from '../../composables/useAppToast';

const route = useRoute();
const { loading, error, getById, ensureLoaded } = useNewsData();
ensureLoaded();

const { toast } = useAppToast();
watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title: 'Не удалось загрузить новость',
      description: String(val),
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const newsId = computed(() => String(route.params.id ?? '').trim());
const item = computed(() => (newsId.value ? getById(newsId.value) : undefined));

const title = computed(() => item.value?.title || (newsId.value ? `Новость #${newsId.value}` : 'Новость'));
const date = computed(() => formatUnixDate(item.value?.timestamp ?? null));
const imageSrc = computed(() => resolveNewsImageSrc(item.value?.announceImagePath ?? null));
const html = computed(() => item.value?.html || item.value?.shortHtml || '');
</script>

<template>
  <UMain class="flex flex-1 min-h-0">
    <UContainer class="flex flex-col gap-4 w-full min-h-0">
      <UCard>
        <template #header>
          <UContainer class="flex flex-col gap-2 p-0">
            <UButton to="/news" variant="ghost" color="neutral" icon="i-lucide-arrow-left" class="w-fit">
              Все новости
            </UButton>
            <div class="flex flex-col gap-1">
              <h1 class="text-2xl font-semibold leading-tight text-highlighted">{{ title }}</h1>
              <div class="flex items-center gap-2 text-sm text-muted">
                <UBadge size="sm" variant="subtle" color="primary" label="Новости" />
                <span v-if="date">{{ date }}</span>
                <USkeleton v-else-if="loading" class="h-4 w-24" />
              </div>
            </div>
          </UContainer>
        </template>

        <UContainer class="flex flex-col gap-4 p-0">
          <UContainer v-if="imageSrc" class="p-0">
            <img
              :src="imageSrc"
              :alt="title"
              class="w-full max-h-[420px] object-cover rounded-lg border border-default"
              loading="lazy"
            />
          </UContainer>

          <USkeleton v-if="loading && !item" class="h-48 w-full" />

          <div
            v-else
            class="prose prose-neutral dark:prose-invert max-w-none"
            v-html="html"
          />

          <UEmpty
            v-if="!loading && !item"
            icon="i-lucide-file-question"
            title="Новость не найдена"
            description="Возможно, она была удалена или ссылка неверная."
          />
        </UContainer>
      </UCard>
    </UContainer>
  </UMain>
</template>

