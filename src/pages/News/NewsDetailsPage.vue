<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { ButtonProps, BlogPostProps } from '@nuxt/ui';
import { useNewsData, formatNewsDate, resolveNewsImageSrc } from '../../composables/useNewsData';
import { useNewsReactions } from '../../composables/useNewsReactions';
import { newsEditorToolbarItems } from '../../composables/newsEditorToolbar';
import { newsEditorExtensions, newsEditorEmojiMenuItems } from '../../composables/newsEditorExtensions';
import { newsEditorHandlers } from '../../composables/newsEditorHandlers';
import { newsEditorSlideoverUi } from '../../composables/newsEditorSlideoverUi';
import { newsEditorHtmlClass } from '../../composables/newsEditorHtmlClass';
import { useAppToast } from '../../composables/useAppToast';
import { useSectionAccess } from '../../composables/useSectionAccess';
import { apiSessionFetch } from '../../composables/useAuthSession';

const route  = useRoute();
const router = useRouter();
const { loading, error, getById, ensureLoaded, sortedNews, reload, patchItem } = useNewsData();
ensureLoaded();

const { toast } = useAppToast();
watch(error, (val) => {
  if (!val) return;
  toast.add({ title: 'Не удалось загрузить новость', description: String(val), color: 'error', icon: 'i-lucide-alert-circle' });
}, { immediate: true });

const newsId = computed(() => String(route.params.id ?? '').trim());
const item   = computed(() => (newsId.value ? getById(newsId.value) : undefined));

const title    = computed(() => item.value?.title || (newsId.value ? `Новость #${newsId.value}` : 'Новость'));
const date     = computed(() => formatNewsDate(item.value?.date ?? null));
const imageSrc = computed(() => resolveNewsImageSrc(item.value?.imagePath ?? null));

const isKiosk      = computed(() => route.matched?.some((r) => r.meta?.kiosk));
const newsListPath = computed(() => (isKiosk.value ? '/kiosk/news' : '/news'));
const { canEditSection, ensureLoaded: ensureSectionAccess } = useSectionAccess();
ensureSectionAccess();
const isAdmin      = computed(() => canEditSection('news'));

const appendEditorEmojiTo = () => document.body;

const surround = computed(() => {
  const id = item.value?.id;
  if (!id) return { prev: null, next: null };
  const list = sortedNews.value;
  const idx  = list.findIndex((x) => x.id === id);
  if (idx < 0) return { prev: null, next: null };
  const prev   = idx > 0 ? list[idx - 1] : null;
  const next   = idx < list.length - 1 ? list[idx + 1] : null;
  const prefix = isKiosk.value ? '/kiosk/news/' : '/news/';
  return {
    prev: prev ? { title: prev.title || `Новость #${prev.id}`, to: `${prefix}${prev.id}` } : null,
    next: next ? { title: next.title || `Новость #${next.id}`, to: `${prefix}${next.id}` } : null,
  };
});

// ─── Likes / Views (в БД) ────────────────────────────────────────────────────

const { isLiked: isLikedFn, toggleLike: toggleLikeAction } = useNewsReactions();

const viewSessionKey = 'news-viewed:v1';

function safeParseJson(raw: string | null): any {
  if (!raw) return null;
  try { return JSON.parse(raw); } catch { return null; }
}

function formatCountRu(n: number): string {
  const v = Number.isFinite(Number(n)) ? Number(n) : 0;
  return Math.max(0, Math.round(v)).toLocaleString('ru-RU');
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
    id:          String(d?.id ?? ''),
    title:       String(d?.title ?? ''),
    category:    String(d?.category ?? ''),
    description: String(d?.description ?? ''),
    date:        String(d?.date ?? ''),
    imagePath:   d?.image_path ?? null,
    createdAt:   d?.created_at ?? null,
    likes: Number(d?.likes ?? 0) || 0,
    views: Number(d?.views ?? 0) || 0,
  };
}

const isLiked = computed(() => item.value ? isLikedFn(item.value.id) : false);

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

// ─── Edit form (по аналогии с EventDetailsPage) ────────────────────────────────

const categoryOptions = [
  { value: 'Новости',     label: 'Новости' },
  { value: 'Мероприятия', label: 'Мероприятия' },
  { value: 'Объявления',  label: 'Объявления' },
  { value: 'Архив',       label: 'Архив' },
];

type EditFormState = {
  title:       string;
  category:    string | undefined;
  description: string;
  date:        string;
};

const editOpen  = ref(false);
const editState = reactive<EditFormState>({
  title:       '',
  category:    undefined,
  description: '',
  date:        '',
});

const editImageFile    = ref<File | null>(null);
const editImagePreview = ref('');

function onEditImageSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  editImageFile.value = file;
  if (editImagePreview.value) URL.revokeObjectURL(editImagePreview.value);
  editImagePreview.value = file ? URL.createObjectURL(file) : '';
}

type SingleDateValue = { value?: any } | any | null;
const editDateValue  = ref<SingleDateValue>(null);
const editSubmitting = ref(false);
const editError      = ref<string | null>(null);

function fillEditState() {
  if (!item.value) return;
  editState.title       = item.value.title       ?? '';
  editState.category    = item.value.category    || undefined;
  editState.description = String(item.value.description ?? '');
  editState.date        = item.value.date        ?? '';
  editImageFile.value   = null;
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
  editError.value      = null;
  try {
    let imagePatch: { image_path?: string } = {};
    if (editImageFile.value) {
      const formData = new FormData();
      formData.append('image', editImageFile.value);
      const uploadRes  = await fetch('/api/Upload/upload.php', { method: 'POST', body: formData });
      const uploadJson = await uploadRes.json();
      if (!uploadJson.success) throw new Error(uploadJson.message || 'Ошибка загрузки изображения');
      imagePatch = { image_path: uploadJson.data.image };
    }

    const json = await apiSessionFetch(`/api/news.php?id=${item.value.id}`, {
      method: 'PUT',
      json: {
        title:       editState.title.trim(),
        category:    editState.category || '',
        description: editState.description.trim(),
        date:        editState.date,
        image_path:  item.value.imagePath,
        ...imagePatch,
      },
    });
    if (!json.success) throw new Error(json.message || 'Ошибка сохранения');

    const data = json.data as any;
    patchItem({
      id:          String(data.id),
      title:       String(data.title       ?? ''),
      category:    String(data.category    ?? ''),
      description: String(data.description ?? ''),
      date:        String(data.date        ?? ''),
      imagePath:   data.image_path ?? null,
      createdAt:   data.created_at ?? null,
      likes:       Number(data.likes ?? 0) || 0,
      views:       Number(data.views ?? 0) || 0,
    });
    editOpen.value = false;
  } catch (e: any) {
    editError.value = e.message ?? 'Ошибка при сохранении';
  } finally {
    editSubmitting.value = false;
  }
}

// ─── Delete ───────────────────────────────────────────────────────────────────

const deleteConfirmOpen = ref(false);
const deleteSubmitting  = ref(false);
const deleteError       = ref<string | null>(null);

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
  deleteError.value      = null;
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

// ─── Лента новостей (Sidebar) ──────────────────────────────────────────────────

const newsLinks = <ButtonProps[]>([
  {
    icon: 'i-lucide-arrow-up-right',
    to: '/news',
    size: 'xl',
    color: 'neutral',
    variant: 'outline',
    class: 'rounded-full',
  },
]);

type NewsItem = {
  id: string;
  likes: number;
  views: number;
  post: BlogPostProps;
};

const newsPageSize = 6;
const visibleNewsCount = ref(newsPageSize);

function newsPreviewText(html: string, maxLen: number): string {
  const plain = String(html ?? '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return plain.length > maxLen ? `${plain.slice(0, maxLen)}…` : plain;
}

const allNewsItems = computed<NewsItem[]>(() =>
  sortedNews.value
    .filter(n => n.id !== item.value?.id) // Исключаем текущую новость
    .map((n) => {
      const imgSrc = resolveNewsImageSrc(n.imagePath);
      return {
        id: n.id,
        likes: n.likes ?? 0,
        views: n.views ?? 0,
        post: {
          title: n.title || `Новость #${n.id}`,
          description: newsPreviewText(n.description, 220),
          image: imgSrc ? { src: imgSrc, alt: n.title || 'Новость' } : { src: '/src/img/Logo.svg', alt: n.title || 'Новость' },
          to: `/news/${n.id}`,
          date: n.date || undefined,
          badge: n.category || 'Новости',
        },
      };
    })
);

const newsItems = computed<NewsItem[]>(() => allNewsItems.value.slice(0, visibleNewsCount.value));
const hasMoreNews = computed(() => visibleNewsCount.value < allNewsItems.value.length);

function showMoreNews() {
  visibleNewsCount.value += newsPageSize;
}
</script>

<template>
  <UMain class="relative flex flex-col xl:flex-row w-full h-full min-h-0 gap-6">
    <div class="flex-1 min-h-0 overflow-y-auto max-w-full w-full scrollbar-hide">

      <div v-if="loading" class="space-y-6">
        <USkeleton class="h-96 rounded-lg" />
        <USkeleton class="h-64 rounded-lg" />
      </div>

      <div v-else-if="item" class="flex flex-col gap-6 p-px">
        <!-- Hero -->
        <div class="relative overflow-hidden rounded-lg ring ring-default bg-default">
          <div class="relative" :class="isKiosk ? 'h-[28rem]' : 'h-96'">
            <img v-if="imageSrc" :src="imageSrc" :alt="title"
              class="absolute inset-0 h-full w-full object-cover" loading="lazy" />
            <div class="absolute inset-0"
              :class="imageSrc ? 'bg-linear-to-t from-black/85 via-black/35 to-transparent'
                               : 'bg-linear-to-br from-primary/12 via-transparent to-primary/8'" />
            <div class="absolute inset-0 flex flex-col justify-end" :class="isKiosk ? 'p-8' : 'p-6'">
              <div class="flex flex-wrap items-center gap-2 mb-2">
                <UBadge v-if="item.category" color="primary" variant="solid" :size="isKiosk ? 'xl' : 'lg'">{{ item.category }}</UBadge>
                <UBadge v-if="date" color="neutral" variant="soft" :size="isKiosk ? 'xl' : 'lg'" class="backdrop-blur">{{ date }}</UBadge>
                <UBadge
                  v-if="!isKiosk"
                  as="button"
                  type="button"
                  :color="isLiked ? 'primary' : 'neutral'"
                  variant="soft"
                  size="lg"
                  leading
                  icon="i-lucide-heart"
                  :label="formatCountRu(item.likes)"
                  :class="[
                    'relative z-10 cursor-pointer shrink-0 [&_svg]:stroke-[1.75]',
                    isLiked
                      ? '[&_svg]:stroke-primary [&_svg_path]:fill-primary [&_svg_path]:stroke-primary'
                      : '[&_svg]:stroke-current [&_svg_path]:fill-none [&_svg_path]:stroke-current',
                  ]"
                  @click.stop.prevent="toggleLike"
                />
                <UBadge color="neutral" variant="soft" :size="isKiosk ? 'xl' : 'lg'" class="backdrop-blur">
                  {{ formatCountRu(item.views) }} просмотров
                </UBadge>
              </div>
              <div class="flex items-end justify-between gap-4 flex-wrap">
                <h1 class="font-semibold tracking-tight min-w-0 flex-1"
                  :class="[
                    imageSrc ? 'text-white' : 'text-highlighted',
                    isKiosk ? 'text-5xl leading-tight' : 'text-2xl sm:text-4xl',
                  ]">
                  {{ title }}
                </h1>
                <div v-if="isAdmin" class="flex gap-2">
                  <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-pencil" @click="openEdit">
                    Изменить
                  </UButton>
                  <UButton color="error" variant="subtle" size="lg" icon="i-lucide-trash-2" @click="deleteConfirmOpen = true">
                    Удалить
                  </UButton>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Description -->
        <UCard v-if="item.description" class="rounded-lg">
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-file-text" :class="isKiosk ? 'size-7 text-primary' : 'size-5 text-primary'" />
              <span :class="isKiosk ? 'text-xl font-semibold text-highlighted' : 'text-sm font-semibold text-highlighted'">Описание</span>
            </div>
          </template>
          <div
            :class="[
              'text-default px-0 sm:px-0 md:px-0 lg:px-0 xl:px-0',
              newsEditorHtmlClass,
              isKiosk ? 'text-3xl [&_p]:leading-relaxed [&_p]:mb-4' : '',
            ]"
            v-html="item.description"
          />
        </UCard>

        <!-- Навигация между новостями -->
        <div
          v-if="surround.prev || surround.next"
          :class="[
            'gap-3 grid w-full',
            isKiosk ? 'grid-cols-2' : 'grid-cols-1 sm:grid-cols-2',
          ]"
        >
          <RouterLink
            v-if="surround.prev"
            :to="surround.prev.to"
            :class="[
              'flex flex-col rounded-lg border border-default hover:bg-elevated/50 transition-colors min-w-0 w-full',
              isKiosk ? 'gap-2 p-6' : 'gap-1 p-4',
            ]"
          >
            <span :class="['flex items-center gap-1.5 text-muted', isKiosk ? 'text-lg' : 'text-xs']">
              <UIcon name="i-lucide-arrow-left" :class="isKiosk ? 'size-5' : 'size-3.5'" />Предыдущая
            </span>
            <span :class="['font-medium text-highlighted line-clamp-2', isKiosk ? 'text-2xl' : 'text-sm']">{{ surround.prev.title }}</span>
          </RouterLink>
          <RouterLink
            v-if="surround.next"
            :to="surround.next.to"
            :class="[
              'flex flex-col rounded-lg border border-default hover:bg-elevated/50 transition-colors text-right items-end min-w-0 w-full',
              !surround.prev ? 'col-start-2' : '',
              isKiosk ? 'gap-2 p-6' : 'gap-1 p-4 sm:text-right sm:items-end',
            ]"
          >
            <span :class="['flex items-center gap-1.5 text-muted', isKiosk ? 'text-lg' : 'text-xs']">
              Следующая<UIcon name="i-lucide-arrow-right" :class="isKiosk ? 'size-5' : 'size-3.5'" />
            </span>
            <span :class="['font-medium text-highlighted line-clamp-2', isKiosk ? 'text-2xl' : 'text-sm']">{{ surround.next.title }}</span>
          </RouterLink>
        </div>
      </div>

      <div v-else class="py-10">
        <UEmpty icon="i-lucide-file-question" title="Новость не найдена"
          description="Возможно, она была удалена или ссылка неверная." />
      </div>
    </div>

    <!-- Правая колонка: Лента новостей -->
    <div class="hidden xl:flex w-96 flex-col gap-3 min-h-0  shrink-0">
            <UScrollArea class="flex-1 min-h-0 min-w-0 scrollbar-hide">
        <div class="flex flex-col gap-3 p-px">
          <UBlogPost
            v-for="sidebarItem in newsItems"
            :key="sidebarItem.id"
            v-bind="sidebarItem.post"
            class="w-full"
          >
            <template #description>
              <p class="text-base text-pretty text-muted">
                {{ sidebarItem.post.description }}
              </p>
              <div
                class="relative z-10 flex justify-between items-center gap-3 w-full mt-2"
                @click.stop
              >
                <UBadge
                  v-if="!isKiosk"
                  as="button"
                  type="button"
                  :color="isLikedFn(sidebarItem.id) ? 'primary' : 'neutral'"
                  variant="soft"
                  size="lg"
                  leading
                  icon="i-lucide-heart"
                  :label="formatCountRu(sidebarItem.likes)"
                  :class="[
                    'relative z-10 cursor-pointer shrink-0 [&_svg]:stroke-[1.75]',
                    isLikedFn(sidebarItem.id)
                      ? '[&_svg]:stroke-primary [&_svg_path]:fill-primary [&_svg_path]:stroke-primary'
                      : '[&_svg]:stroke-current [&_svg_path]:fill-none [&_svg_path]:stroke-current',
                  ]"
                  @click.stop.prevent="toggleLikeAction(sidebarItem.id)"
                />
                <span class="shrink-0 text-sm text-muted">
                  {{ formatCountRu(sidebarItem.views) }} просмотров
                </span>
              </div>
            </template>
          </UBlogPost>

          <UButton
            v-if="hasMoreNews"
            type="button"
            color="neutral"
            variant="outline"
            size="xl"
            class="relative z-10 w-full shrink-0 justify-center"
            icon="i-lucide-chevron-down"
            @click="showMoreNews"
          >
            Показать ещё
          </UButton>
        </div>
      </UScrollArea>
    </div>

    <!-- Slideover редактирования -->
    <USlideover
      v-model:open="editOpen"
      side="right"
      title="Редактирование новости"
      description="Измените данные новости"
      :ui="{
        content:
          '!max-w-full sm:!max-w-2xl lg:!max-w-4xl xl:!max-w-5xl',
      }"
    >
      <template #body>
        <div class="flex flex-col gap-4 py-2">
          <UForm :state="editState" class="space-y-4" @submit.prevent="handleEditSubmit">
            <UFormField label="Название" name="title" required>
              <UInput v-model="editState.title" size="lg" placeholder="Заголовок новости" class="w-full" />
            </UFormField>

            <UFormField label="Категория" name="category">
              <USelect
                v-model="editState.category"
                :items="categoryOptions"
                :placeholder="undefined"
                size="lg"
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
              <UInputDate v-model="editDateValue" size="lg" class="w-full">
                <template #trailing>
                  <UPopover>
                    <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar"
                      aria-label="Выбрать дату" class="px-0" />
                    <template #content>
                      <UCalendar v-model="editDateValue" class="p-2" />
                    </template>
                  </UPopover>
                </template>
              </UInputDate>
            </UFormField>

            <UFormField label="Изображение" name="image">
              <div class="flex flex-col gap-2 w-full">
                <input id="editFileInput" type="file" accept="image/jpeg,image/png,image/webp"
                  class="hidden" @change="onEditImageSelected" />
                <img v-if="editImagePreview" :src="editImagePreview" alt="Предпросмотр"
                  class="w-full h-40 object-cover rounded-lg" />
                <label for="editFileInput"
                  class="flex items-center justify-center gap-2 cursor-pointer rounded-lg border border-default px-4 py-2.5 text-sm font-medium text-default hover:bg-elevated/50 transition-colors">
                  <UIcon name="i-lucide-upload" class="text-base shrink-0" />
                  {{ editImageFile ? 'Изменить фото' : 'Заменить фото' }}
                </label>
                <p v-if="editImageFile" class="text-xs text-muted truncate px-1">{{ editImageFile.name }}</p>
              </div>
            </UFormField>

            <UAlert v-if="editError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="editError" />
          </UForm>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="editOpen = false">Отмена</UButton>
          <UButton size="xl" class="w-full justify-center" :loading="editSubmitting" @click="handleEditSubmit">Сохранить</UButton>
        </div>
      </template>
    </USlideover>

    <!-- Модал подтверждения удаления -->
    <UModal v-model:open="deleteConfirmOpen" title="Удалить новость?" description="Подтвердите удаление">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить <strong>«{{ item?.title }}»</strong>? Это действие нельзя отменить.
        </p>
        <UAlert v-if="deleteError" color="error" variant="subtle" icon="i-lucide-alert-circle"
          :description="deleteError" class="mt-3" />
      </template>
      <template #footer>
        <div class="flex gap-3 justify-end w-full">
          <UButton color="neutral" variant="outline" size="lg" @click="deleteConfirmOpen = false">Отмена</UButton>
          <UButton color="red" size="lg" :loading="deleteSubmitting" @click="handleDelete">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>