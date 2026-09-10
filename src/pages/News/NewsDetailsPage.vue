<script setup lang="ts">
import { computed, nextTick, onUnmounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useNewsData, formatNewsDate, resolveNewsImageSrc } from '../../composables/useNewsData';
import { useNewsReactions } from '../../composables/useNewsReactions';
import { newsEditorToolbarItems } from '../../composables/newsEditorToolbar';
import { newsEditorExtensions } from '../../composables/newsEditorExtensions';
import { newsEditorHandlers } from '../../composables/newsEditorHandlers';
import { newsEditorSlideoverUi } from '../../composables/newsEditorSlideoverUi';
import { newsEditorHtmlClass } from '../../composables/newsEditorHtmlClass';
import { useAppToast } from '../../composables/useAppToast';
import { useSectionAccess } from '../../composables/useSectionAccess';
import { apiSessionFetch } from '../../composables/useAuthSession';
import { useBreadcrumbCurrentLabel } from '../../composables/usePortalNavigation';

const route = useRoute();
const router = useRouter();
const { loading, error, getById, ensureLoaded, sortedNews, reload, patchItem } = useNewsData();
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

const newsScrollEl = ref<HTMLElement | null>(null);

watch(newsId, async () => {
  await nextTick();
  const el = newsScrollEl.value;
  if (el) el.scrollTo({ top: 0, behavior: 'smooth' });
  else window.scrollTo({ top: 0, behavior: 'smooth' });
});

const title = computed(() => item.value?.title || (newsId.value ? `Новость #${newsId.value}` : 'Новость'));
const date = computed(() => formatNewsDate(item.value?.date ?? null));
const imageSrc = computed(() => resolveNewsImageSrc(item.value?.imagePath ?? null));

const breadcrumbLabel = useBreadcrumbCurrentLabel();
watch(
  title,
  (t) => {
    breadcrumbLabel.value = item.value ? t.trim() || null : null;
  },
  { immediate: true },
);
onUnmounted(() => {
  breadcrumbLabel.value = null;
});

const isKiosk = computed(() => route.matched?.some((r) => r.meta?.kiosk));
const newsListPath = computed(() => (isKiosk.value ? '/kiosk/news' : '/news'));
const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const isAdmin = computed(() => canEditSection('news'));

function onKioskNewsBodyClick(e: MouseEvent) {
  if (!isKiosk.value) return;
  const target = e.target as HTMLElement | null;
  const link = target?.closest?.('a');
  if (!link) return;
  e.preventDefault();
  e.stopPropagation();
}

const surround = computed(() => {
  const id = item.value?.id;
  if (!id) return { prev: null, next: null };
  const list = sortedNews.value;
  const idx = list.findIndex((x) => x.id === id);
  if (idx < 0) return { prev: null, next: null };
  const prev = idx > 0 ? list[idx - 1] : null;
  const next = idx < list.length - 1 ? list[idx + 1] : null;
  const prefix = isKiosk.value ? '/kiosk/news/' : '/news/';
  return {
    prev: prev ? { title: prev.title || `Новость #${prev.id}`, to: `${prefix}${prev.id}` } : null,
    next: next ? { title: next.title || `Новость #${next.id}`, to: `${prefix}${next.id}` } : null,
  };
});

const relatedNews = computed(() => {
  const id = item.value?.id;
  if (!id) return [];
  const prefix = isKiosk.value ? '/kiosk/news/' : '/news/';
  const category = String(item.value?.category ?? '').trim().toLowerCase();
  const others = sortedNews.value.filter((n) => n.id !== id);
  const sameCategory = category
    ? others.filter((n) => String(n.category ?? '').trim().toLowerCase() === category)
    : [];
  const pool = sameCategory.length ? sameCategory : others;
  return pool.slice(0, 6).map((n) => ({
    id: n.id,
    title: n.title || `Новость #${n.id}`,
    date: formatNewsDate(n.date) || '',
    to: `${prefix}${n.id}`,
    coverSrc: resolveNewsImageSrc(n.imagePath) || '',
    views: n.views ?? 0,
  }));
});

const { isLiked: isLikedFn, toggleLike: toggleLikeAction } = useNewsReactions();

const viewSessionKey = 'news-viewed:v1';

function safeParseJson(raw: string | null): any {
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

function formatCountRu(n: number): string {
  const v = Number.isFinite(Number(n)) ? Number(n) : 0;
  return Math.max(0, Math.round(v)).toLocaleString('ru-RU');
}

function viewsLabel(n: number): string {
  const v = Math.max(0, Math.round(Number(n) || 0));
  const mod10 = v % 10;
  const mod100 = v % 100;
  const formatted = formatCountRu(v);
  if (mod10 === 1 && mod100 !== 11) return `${formatted} просмотр`;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return `${formatted} просмотра`;
  return `${formatted} просмотров`;
}

function readViewedMap(): Record<string, boolean> {
  if (typeof window === 'undefined') return {};
  return safeParseJson(window.sessionStorage.getItem(viewSessionKey)) ?? {};
}

function writeViewedMap(map: Record<string, boolean>) {
  if (typeof window === 'undefined') return;
  window.sessionStorage.setItem(viewSessionKey, JSON.stringify(map));
}

function mapApiToNewsRecord(d: any) {
  return {
    id: String(d?.id ?? ''),
    title: String(d?.title ?? ''),
    category: String(d?.category ?? ''),
    description: String(d?.description ?? ''),
    date: String(d?.date ?? ''),
    imagePath: d?.image_path ?? null,
    createdAt: d?.created_at ?? null,
    likes: Number(d?.likes ?? 0) || 0,
    views: Number(d?.views ?? 0) || 0,
  };
}

const isLiked = computed(() => (item.value ? isLikedFn(item.value.id) : false));

function toggleLike() {
  if (item.value?.id) void toggleLikeAction(item.value.id);
}

async function incrementViewOnce(id: string) {
  if (typeof window === 'undefined') return;

  const viewed = readViewedMap();
  if (viewed?.[id]) return;

  const before = item.value && { ...item.value };
  if (before) patchItem({ ...before, views: (before.views ?? 0) + 1 });

  try {
    const res = await fetch(`/api/news.php?id=${id}&action=view`, { method: 'POST' });
    const json = await res.json();
    if (!json?.success) throw new Error(json?.message || 'Ошибка обновления просмотров');

    patchItem(mapApiToNewsRecord(json.data));
    viewed[id] = true;
    writeViewedMap(viewed);
  } catch {
    if (before) patchItem(before);
  }
}

watch(
  () => item.value?.id,
  (id) => {
    if (!id) return;
    void incrementViewOnce(id);
  },
  { immediate: true },
);

const categoryOptions = [
  { value: 'Новости', label: 'Новости' },
  { value: 'Мероприятия', label: 'Мероприятия' },
  { value: 'Объявления', label: 'Объявления' },
  { value: 'Архив', label: 'Архив' },
];

type EditFormState = {
  title: string;
  category: string | undefined;
  description: string;
  date: string;
};

const editOpen = ref(false);
const editState = reactive<EditFormState>({
  title: '',
  category: undefined,
  description: '',
  date: '',
});

const editImageFile = ref<File | null>(null);
const editImagePreview = ref('');

function onEditImageSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  editImageFile.value = file;
  if (editImagePreview.value) URL.revokeObjectURL(editImagePreview.value);
  editImagePreview.value = file ? URL.createObjectURL(file) : '';
}

type SingleDateValue = { value?: any } | any | null;
const editDateValue = ref<SingleDateValue>(null);
const editSubmitting = ref(false);
const editError = ref<string | null>(null);

function fillEditState() {
  if (!item.value) return;
  editState.title = item.value.title ?? '';
  editState.category = item.value.category || undefined;
  editState.description = String(item.value.description ?? '');
  editState.date = item.value.date ?? '';
  editImageFile.value = null;
  editImagePreview.value = '';
}

function openEdit() {
  fillEditState();
  editOpen.value = true;
}

watch(editDateValue, (val) => {
  const d = val?.value ?? val;
  editState.date = d && typeof d.toString === 'function' ? d.toString() : '';
});

function validateEdit(): boolean {
  if (!editState.title.trim()) {
    editError.value = 'Заполните название.';
    return false;
  }
  if (!editState.date) {
    editError.value = 'Выберите дату.';
    return false;
  }
  editError.value = null;
  return true;
}

async function handleEditSubmit() {
  if (!validateEdit() || !item.value?.id) return;
  editSubmitting.value = true;
  editError.value = null;
  try {
    let imagePatch: { image_path?: string } = {};
    if (editImageFile.value) {
      const formData = new FormData();
      formData.append('image', editImageFile.value);
      const uploadRes = await fetch('/api/Upload/upload.php', { method: 'POST', body: formData });
      const uploadJson = await uploadRes.json();
      if (!uploadJson.success) throw new Error(uploadJson.message || 'Ошибка загрузки изображения');
      imagePatch = { image_path: uploadJson.data.image };
    }

    const json = await apiSessionFetch(`/api/news.php?id=${item.value.id}`, {
      method: 'PUT',
      json: {
        title: editState.title.trim(),
        category: editState.category || '',
        description: editState.description.trim(),
        date: editState.date,
        image_path: item.value.imagePath,
        ...imagePatch,
      },
    });
    if (!json.success) throw new Error(json.message || 'Ошибка сохранения');

    const data = json.data as any;
    patchItem({
      id: String(data.id),
      title: String(data.title ?? ''),
      category: String(data.category ?? ''),
      description: String(data.description ?? ''),
      date: String(data.date ?? ''),
      imagePath: data.image_path ?? null,
      createdAt: data.created_at ?? null,
      likes: Number(data.likes ?? 0) || 0,
      views: Number(data.views ?? 0) || 0,
    });
    editOpen.value = false;
    toast.add({ title: 'Новость обновлена', color: 'success', icon: 'i-lucide-check-circle' });
  } catch (e: any) {
    editError.value = e.message ?? 'Ошибка при сохранении';
  } finally {
    editSubmitting.value = false;
  }
}

const deleteConfirmOpen = ref(false);
const deleteSubmitting = ref(false);
const deleteError = ref<string | null>(null);

watch(editError, (val) => {
  if (!val) return;
  toast.add({ title: 'Не удалось сохранить', description: String(val), color: 'error', icon: 'i-lucide-alert-circle' });
});
watch(deleteError, (val) => {
  if (!val) return;
  toast.add({ title: 'Не удалось удалить', description: String(val), color: 'error', icon: 'i-lucide-alert-circle' });
});

async function handleDelete() {
  if (!item.value?.id) return;
  deleteSubmitting.value = true;
  deleteError.value = null;
  try {
    const json = await apiSessionFetch(`/api/news.php?id=${item.value.id}`, { method: 'DELETE' });
    if (!json.success) throw new Error(json.message || 'Ошибка удаления');
    deleteConfirmOpen.value = false;
    await reload();
    void router.push(newsListPath.value);
  } catch (e: any) {
    deleteError.value = e.message ?? 'Ошибка при удалении';
  } finally {
    deleteSubmitting.value = false;
  }
}

const moreMenuItems = computed(() => {
  if (!isAdmin.value) return [];
  return [
    [
      {
        label: 'Редактировать',
        icon: 'i-lucide-pencil',
        onSelect: openEdit,
      },
      {
        label: 'Удалить новость',
        icon: 'i-lucide-trash-2',
        color: 'error' as const,
        onSelect: () => {
          deleteError.value = null;
          deleteConfirmOpen.value = true;
        },
      },
    ],
  ];
});

const imageLightboxOpen = ref(false);

function downloadCoverImage(e?: Event) {
  e?.preventDefault();
  e?.stopPropagation();
  if (!imageSrc.value) return;
  const a = document.createElement('a');
  a.href = imageSrc.value;
  a.download = `news-${item.value?.id || 'cover'}`;
  a.target = '_blank';
  a.rel = 'noopener';
  a.click();
}

function openCoverLightbox() {
  if (!imageSrc.value) return;
  imageLightboxOpen.value = true;
}

async function removeCoverImage(e?: Event) {
  e?.preventDefault();
  e?.stopPropagation();
  if (!item.value?.id || !imageSrc.value) return;
  try {
    const json = await apiSessionFetch(`/api/news.php?id=${item.value.id}`, {
      method: 'PUT',
      json: {
        title: item.value.title,
        category: item.value.category || '',
        description: item.value.description,
        date: item.value.date,
        image_path: null,
      },
    });
    if (!json.success) throw new Error(json.message || 'Ошибка удаления фото');
    const data = json.data as any;
    patchItem({
      id: String(data.id),
      title: String(data.title ?? ''),
      category: String(data.category ?? ''),
      description: String(data.description ?? ''),
      date: String(data.date ?? ''),
      imagePath: data.image_path ?? null,
      createdAt: data.created_at ?? null,
      likes: Number(data.likes ?? 0) || 0,
      views: Number(data.views ?? 0) || 0,
    });
    toast.add({ title: 'Фото удалено', color: 'success', icon: 'i-lucide-check-circle' });
  } catch (err: any) {
    toast.add({
      title: 'Не удалось удалить фото',
      description: err?.message,
      color: 'error',
      icon: 'i-lucide-alert-circle',
    });
  }
}

function onExtraReaction() {
  toast.add({
    title: 'Реакции',
    description: 'Дополнительные реакции появятся после поддержки на сервере. Пока доступен лайк.',
    color: 'neutral',
    icon: 'i-lucide-smile',
  });
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div
      ref="newsScrollEl"
      class="flex flex-col w-full h-full min-h-0 gap-6 overflow-y-auto scrollbar-hide p-px"
    >
      <div class="flex flex-col gap-6 w-full max-w-[1600px] mx-auto pb-8">
        <div v-if="loading" class="flex flex-col gap-6">
          <USkeleton class="h-12 w-2/3 rounded-lg" />
          <USkeleton class="h-5 w-1/3 rounded-lg" />
          <USkeleton class="h-80 w-full rounded-panel" />
          <USkeleton class="h-40 w-full rounded-lg" />
        </div>

        <template v-else-if="item">
          <div class="grid w-full min-w-0 grid-cols-1 xl:grid-cols-[minmax(0,1fr)_420px] gap-6 items-start">
            <div class="flex min-w-0 w-full flex-col gap-6">
              <div class="flex flex-col gap-3">
                <UPageHeader
                  headline="Новости"
                  :title="title"
                >
                  <template v-if="!isKiosk && moreMenuItems.length" #links>
                    <UDropdownMenu :items="moreMenuItems">
                      <UButton
                        color="neutral"
                        variant="outline"
                        size="md"
                        icon="i-lucide-ellipsis"
                        square
                        aria-label="Ещё действия"
                      />
                    </UDropdownMenu>
                  </template>
                </UPageHeader>

                <div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-muted">
                  <UBadge v-if="item.category" color="info" variant="subtle">
                    {{ item.category }}
                  </UBadge>
                  <span v-if="date" class="inline-flex items-center gap-1.5">
                    <UIcon name="i-lucide-calendar" class="size-4 shrink-0" />
                    {{ date }}
                  </span>
                  <span class="inline-flex items-center gap-1.5">
                    <UIcon name="i-lucide-eye" class="size-4 shrink-0" />
                    {{ viewsLabel(item.views) }}
                  </span>
                  <div
                    v-if="!isKiosk"
                    class="flex flex-wrap items-center gap-1.5"
                  >
                    <UButton
                      type="button"
                      size="xs"
                      :color="isLiked ? 'primary' : 'neutral'"
                      variant="subtle"
                      :label="formatCountRu(item.likes)"
                      @click="toggleLike"
                    >
                      <template #leading>
                        <span class="text-sm leading-none" aria-hidden="true">👍</span>
                      </template>
                    </UButton>
                    <UButton
                      type="button"
                      size="xs"
                      color="neutral"
                      variant="subtle"
                      square
                      aria-label="Нравится"
                      @click="onExtraReaction"
                    >
                      <span class="text-sm leading-none" aria-hidden="true">❤️</span>
                    </UButton>
                    <UButton
                      type="button"
                      size="xs"
                      color="neutral"
                      variant="subtle"
                      square
                      aria-label="Улыбка"
                      @click="onExtraReaction"
                    >
                      <span class="text-sm leading-none" aria-hidden="true">🙂</span>
                    </UButton>
                    <UButton
                      type="button"
                      size="xs"
                      color="neutral"
                      variant="ghost"
                      square
                      icon="i-lucide-plus"
                      aria-label="Добавить реакцию"
                      @click="onExtraReaction"
                    />
                  </div>
                </div>
              </div>

              <div
                v-if="imageSrc"
                class="group relative h-80 overflow-hidden rounded-panel bg-elevated"
              >
                <img
                  :src="imageSrc"
                  :alt="title"
                  class="size-full object-cover"
                  loading="lazy"
                />
                <div
                  class="pointer-events-none absolute inset-0 opacity-0 group-hover:opacity-100 transition"
                >
                  <div class="absolute top-2 right-2 flex gap-1 pointer-events-auto">
                    <UButton
                      type="button"
                      color="neutral"
                      variant="solid"
                      size="xs"
                      icon="i-lucide-download"
                      square
                      class="bg-black/60 hover:bg-black/80 text-white ring-0"
                      aria-label="Скачать"
                      @click="downloadCoverImage($event)"
                    />
                    <UButton
                      type="button"
                      color="neutral"
                      variant="solid"
                      size="xs"
                      icon="i-lucide-expand"
                      square
                      class="bg-black/60 hover:bg-black/80 text-white ring-0"
                      aria-label="Открыть"
                      @click="openCoverLightbox"
                    />
                    <UButton
                      v-if="isAdmin && !isKiosk"
                      type="button"
                      color="error"
                      variant="solid"
                      size="xs"
                      icon="i-lucide-trash-2"
                      square
                      class="bg-black/60 hover:bg-error text-white ring-0"
                      aria-label="Удалить"
                      @click="removeCoverImage($event)"
                    />
                  </div>
                </div>
              </div>

              <div
                v-if="item.description"
                :class="[
                  'w-full text-default',
                  newsEditorHtmlClass,
                  isKiosk ? 'text-3xl [&_p]:leading-relaxed [&_p]:mb-4 kiosk-news-body' : 'text-base leading-relaxed',
                ]"
                v-html="item.description"
                @click.capture="onKioskNewsBodyClick"
              />
              <div
                v-if="surround.prev || surround.next"
                class="grid grid-cols-1 sm:grid-cols-2 gap-3 w-full"
              >
                <RouterLink
                  v-if="surround.prev"
                  :to="surround.prev.to"
                  class="flex flex-col gap-1 rounded-panel p-4 ring-1 ring-default hover:ring-primary transition min-w-0"
                >
                  <span class="flex items-center gap-1.5 text-xs text-muted">
                    <UIcon name="i-lucide-arrow-left" class="size-3.5" />
                    Предыдущая
                  </span>
                  <span class="text-sm font-medium text-highlighted line-clamp-2">
                    {{ surround.prev.title }}
                  </span>
                </RouterLink>
                <RouterLink
                  v-if="surround.next"
                  :to="surround.next.to"
                  class="flex flex-col gap-1 rounded-panel p-4 ring-1 ring-default hover:ring-primary transition text-right items-end min-w-0"
                  :class="!surround.prev ? 'sm:col-start-2' : ''"
                >
                  <span class="flex items-center gap-1.5 text-xs text-muted">
                    Следующая
                    <UIcon name="i-lucide-arrow-right" class="size-3.5" />
                  </span>
                  <span class="text-sm font-medium text-highlighted line-clamp-2">
                    {{ surround.next.title }}
                  </span>
                </RouterLink>
              </div>
            </div>

            <aside
              v-if="relatedNews.length"
              class="w-full min-w-0 flex flex-col gap-4 xl:sticky xl:top-0"
              aria-label="Новости по теме"
            >
              <div class="flex items-center justify-between gap-2">
                <h2 class="text-lg font-semibold text-highlighted">По теме</h2>
                <UButton
                  :to="newsListPath"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  icon="i-lucide-arrow-up-right"
                  square
                  aria-label="Все новости"
                />
              </div>

              <div class="flex flex-col gap-3">
                <RouterLink
                  v-for="post in relatedNews"
                  :key="post.id"
                  :to="post.to"
                  class="group flex gap-3 rounded-panel p-px transition ring-1 ring-transparent hover:ring-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                >
                  <div class="relative size-20 shrink-0 overflow-hidden rounded-panel bg-elevated">
                    <img
                      v-if="post.coverSrc"
                      :src="post.coverSrc"
                      :alt="post.title"
                      loading="lazy"
                      decoding="async"
                      class="size-full object-cover transition duration-300 group-hover:scale-[1.02]"
                    />
                    <div
                      v-else
                      class="size-full flex items-center justify-center text-muted"
                    >
                      <UIcon name="i-lucide-newspaper" class="size-5" />
                    </div>
                  </div>
                  <div class="flex min-w-0 flex-1 flex-col gap-1 py-0.5">
                    <p v-if="post.date" class="text-xs text-muted">{{ post.date }}</p>
                    <h3 class="text-sm font-semibold text-highlighted line-clamp-2 text-pretty">
                      {{ post.title }}
                    </h3>
                    <span class="mt-auto inline-flex items-center gap-1 text-xs text-muted tabular-nums">
                      <UIcon name="i-lucide-eye" class="size-3.5 shrink-0" />
                      {{ viewsLabel(post.views) }}
                    </span>
                  </div>
                </RouterLink>
              </div>
            </aside>
          </div>
        </template>

        <UEmpty
          v-else
          variant="naked"
          icon="i-lucide-file-question"
          title="Новость не найдена"
          description="Возможно, она была удалена или ссылка неверная."
          class="w-full py-12"
        />
      </div>
    </div>

    <UModal
      v-model:open="imageLightboxOpen"
      class="p-0"
      :ui="{ content: 'bg-transparent shadow-none ring-0 w-auto max-w-[95vw]', header: 'hidden', body: 'p-0' }"
    >
      <template #content>
        <div v-if="imageSrc" class="flex flex-col items-center justify-center gap-3 p-0">
          <img
            :src="imageSrc"
            :alt="title"
            decoding="async"
            class="block w-auto h-auto max-w-[95vw] max-h-[85vh] rounded-lg"
          />
        </div>
      </template>
    </UModal>

    <USlideover
      v-model:open="editOpen"
      side="right"
      title="Редактирование новости"
      description="Измените данные новости"
      :ui="{
        content: '!max-w-full sm:!max-w-2xl lg:!max-w-4xl xl:!max-w-5xl',
      }"
    >
      <template #body>
        <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
          <UFormField label="Название" name="title" required>
            <UInput v-model="editState.title" size="xl" placeholder="Заголовок новости" class="w-full" />
          </UFormField>

          <UFormField label="Категория" name="category">
            <USelect
              v-model="editState.category"
              :items="categoryOptions"
              size="xl"
              class="w-full"
            />
          </UFormField>

          <UFormField label="Описание" name="description">
            <UEditor
              v-slot="{ editor }"
              v-model="editState.description"
              content-type="html"
              :extensions="newsEditorExtensions"
              :handlers="newsEditorHandlers"
              :ui="newsEditorSlideoverUi"
              placeholder="Текст новости…"
              class="w-full min-h-56 rounded-lg border border-default overflow-hidden"
            >
              <UEditorToolbar
                :editor="editor"
                :items="newsEditorToolbarItems"
                class="sticky top-0 z-10 border-b border-default bg-default/95 backdrop-blur-sm px-2 py-1.5 overflow-x-auto"
              />
            </UEditor>
          </UFormField>

          <UFormField label="Дата" name="date" required>
            <UInputDate v-model="editDateValue" size="xl" class="w-full">
              <template #trailing>
                <UPopover>
                  <UButton
                    color="neutral"
                    variant="link"
                    size="md"
                    icon="i-lucide-calendar"
                    aria-label="Выбрать дату"
                    class="px-0"
                  />
                  <template #content>
                    <UCalendar v-model="editDateValue" class="p-2" />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </UFormField>

          <UFormField label="Изображение" name="image">
            <div class="flex flex-col gap-2 w-full">
              <input
                id="editFileInput"
                type="file"
                accept="image/jpeg,image/png,image/webp"
                class="hidden"
                @change="onEditImageSelected"
              />
              <img
                v-if="editImagePreview"
                :src="editImagePreview"
                alt="Предпросмотр"
                class="w-full h-40 object-cover rounded-lg"
              />
              <label
                for="editFileInput"
                class="flex items-center justify-center gap-2 cursor-pointer rounded-lg border border-default px-4 py-2.5 text-sm font-medium text-default hover:bg-elevated/50 transition-colors"
              >
                <UIcon name="i-lucide-upload" class="text-base shrink-0" />
                {{ editImageFile ? 'Изменить фото' : 'Заменить фото' }}
              </label>
              <p v-if="editImageFile" class="text-xs text-muted truncate px-1">
                {{ editImageFile.name }}
              </p>
            </div>
          </UFormField>

          <UAlert
            v-if="editError"
            color="error"
            variant="subtle"
            icon="i-lucide-alert-circle"
            :description="editError"
          />
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton
            color="neutral"
            variant="outline"
            size="md"
            class="w-full justify-center"
            @click="editOpen = false"
          >
            Отмена
          </UButton>
          <UButton
            size="md"
            class="w-full justify-center"
            :loading="editSubmitting"
            @click="handleEditSubmit"
          >
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>

    <UModal v-model:open="deleteConfirmOpen" title="Удалить новость?" description="Подтвердите удаление">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить <strong>«{{ item?.title }}»</strong>? Это действие нельзя отменить.
        </p>
        <UAlert
          v-if="deleteError"
          color="error"
          variant="subtle"
          icon="i-lucide-alert-circle"
          :description="deleteError"
          class="mt-3"
        />
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton color="neutral" variant="outline" size="md" @click="deleteConfirmOpen = false">
            Отмена
          </UButton>
          <UButton color="error" size="md" :loading="deleteSubmitting" @click="handleDelete">
            Удалить
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>

<style scoped>
.kiosk-news-body :deep(a) {
  pointer-events: none;
  cursor: default;
  text-decoration: none;
}
</style>
