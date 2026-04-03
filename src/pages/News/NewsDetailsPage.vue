<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useNewsData, formatNewsDate, resolveNewsImageSrc } from '../../composables/useNewsData';
import { newsEditorToolbarItems } from '../../composables/newsEditorToolbar';
import { newsEditorExtensions, newsEditorEmojiMenuItems } from '../../composables/newsEditorExtensions';
import { newsEditorHandlers } from '../../composables/newsEditorHandlers';
import { newsEditorSlideoverUi } from '../../composables/newsEditorSlideoverUi';
import { newsEditorHtmlClass } from '../../composables/newsEditorHtmlClass';
import { useAppToast } from '../../composables/useAppToast';
import { currentRole } from '../../stores/role';

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
const isAdmin      = computed(() => currentRole.value === 'admin');

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

const viewSessionKey = 'news-viewed:v1';
const isLiked = ref(false);
const likeSubmitting = ref(false);

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

async function incrementViewOnce(newsId: string) {
  if (typeof window === 'undefined') return;

  const viewed = readViewedMap();
  if (viewed?.[newsId]) return;

  const before = item.value && { ...item.value };
  if (before) {
    patchItem({
      ...before,
      views: (before.views ?? 0) + 1,
    });
  }

  try {
    const res = await fetch(`/api/news.php?id=${newsId}&action=view`, { method: 'POST' });
    const json = await res.json();
    if (!json?.success) throw new Error(json?.message || 'Ошибка обновления просмотров');

    // Обновляем UI и только после успеха фиксируем "уже учтено"
    patchItem(mapApiToNewsRecord(json.data));
    viewed[newsId] = true;
    writeViewedMap(viewed);
  } catch {
    // Если запись в БД не удалась — откатываем локальное значение.
    if (before) {
      patchItem(before);
    }
  }
}

async function toggleLike() {
  if (likeSubmitting.value) return;
  const newsId = item.value?.id;
  if (!newsId) return;

  const nextLiked = !isLiked.value;
  const before = item.value && { ...item.value };
  likeSubmitting.value = true;

  try {
    const res = await fetch(`/api/news.php?id=${newsId}&action=like`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ liked: nextLiked }),
    });
    const json = await res.json();
    if (!json?.success) throw new Error(json?.message || 'Ошибка обновления лайка');

    patchItem(mapApiToNewsRecord(json.data));
    isLiked.value = nextLiked;
  } catch (e) {
    if (before) {
      patchItem(before);
    }
  } finally {
    likeSubmitting.value = false;
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

    const res  = await fetch(`/api/news.php?id=${item.value.id}`, {
      method:  'PUT',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({
        title:       editState.title.trim(),
        category:    editState.category || '',
        description: editState.description.trim(),
        date:        editState.date,
        image_path:  item.value.imagePath,
        ...imagePatch,
      }),
    });
    const json = await res.json();
    if (!json.success) throw new Error(json.message || 'Ошибка сохранения');

    patchItem({
      id:          String(json.data.id),
      title:       String(json.data.title       ?? ''),
      category:    String(json.data.category    ?? ''),
      description: String(json.data.description ?? ''),
      date:        String(json.data.date        ?? ''),
      imagePath:   json.data.image_path ?? null,
      createdAt:   json.data.created_at ?? null,
      likes:       Number(json.data.likes ?? 0) || 0,
      views:       Number(json.data.views ?? 0) || 0,
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
    const res  = await fetch(`/api/news.php?id=${item.value.id}`, { method: 'DELETE' });
    const json = await res.json();
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
</script>

<template>
  <UMain class="relative flex flex-col w-full h-full min-h-0 gap-0">
    <div class="flex-1 min-h-0 overflow-y-auto max-w-full w-full scrollbar-hide pb-8">

      <div v-if="loading" class="space-y-6">
        <USkeleton class="h-96 rounded-lg" />
        <USkeleton class="h-64 rounded-lg" />
      </div>

      <div v-else-if="item" class="flex flex-col gap-6 p-px">
        <!-- Hero -->
        <div class="relative overflow-hidden rounded-lg ring ring-default bg-default">
          <div class="relative h-96">
            <img v-if="imageSrc" :src="imageSrc" :alt="title"
              class="absolute inset-0 h-full w-full object-cover" loading="lazy" />
            <div class="absolute inset-0"
              :class="imageSrc ? 'bg-linear-to-t from-black/85 via-black/35 to-transparent'
                               : 'bg-linear-to-br from-primary/12 via-transparent to-primary/8'" />
            <div class="absolute inset-0 p-6 flex flex-col justify-end">
              <div class="flex flex-wrap items-center gap-2 mb-2">
                <UBadge v-if="item.category" color="primary" variant="solid" size="lg">{{ item.category }}</UBadge>
                <UBadge v-if="date" color="neutral" variant="soft" size="lg" class="backdrop-blur">{{ date }}</UBadge>
                <UBadge
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
                <UBadge color="neutral" variant="soft" size="lg" class="backdrop-blur">
                  {{ formatCountRu(item.views) }} просмотров
                </UBadge>
              </div>
              <div class="flex items-end justify-between gap-4 flex-wrap">
                <h1 class="text-2xl sm:text-4xl font-semibold tracking-tight min-w-0 flex-1"
                  :class="imageSrc ? 'text-white' : 'text-highlighted'">
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
              <UIcon name="i-lucide-file-text" class="size-5 text-primary" />
              <span class="text-sm font-semibold text-highlighted">Описание</span>
            </div>
          </template>
          <div :class="['text-default', newsEditorHtmlClass]" class="px-0 sm:px-0 md:px-0 lg:px-0 xl:px-0" v-html="item.description" />
        </UCard>

        <!-- Навигация между новостями -->
        <div v-if="surround.prev || surround.next" class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <RouterLink v-if="surround.prev" :to="surround.prev.to"
            class="flex flex-col gap-1 rounded-lg border border-default p-4 hover:bg-elevated/50 transition-colors">
            <span class="flex items-center gap-1.5 text-xs text-muted">
              <UIcon name="i-lucide-arrow-left" class="size-3.5" />Предыдущая
            </span>
            <span class="text-sm font-medium text-highlighted line-clamp-2">{{ surround.prev.title }}</span>
          </RouterLink>
          <RouterLink v-if="surround.next" :to="surround.next.to"
            class="flex flex-col gap-1 rounded-lg border border-default p-4 hover:bg-elevated/50 transition-colors sm:text-right sm:items-end">
            <span class="flex items-center gap-1.5 text-xs text-muted">
              Следующая<UIcon name="i-lucide-arrow-right" class="size-3.5" />
            </span>
            <span class="text-sm font-medium text-highlighted line-clamp-2">{{ surround.next.title }}</span>
          </RouterLink>
        </div>
      </div>

      <div v-else class="py-10">
        <UEmpty icon="i-lucide-file-question" title="Новость не найдена"
          description="Возможно, она была удалена или ссылка неверная." />
      </div>
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