<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useNewsData, formatNewsDate, resolveNewsImageSrc } from '../../composables/useNewsData';
import { useAppToast } from '../../composables/useAppToast';
import UContentSurround from '../../components/UContentSurround.vue';
import { currentRole } from '../../stores/role';

const route  = useRoute();
const router = useRouter();
const { loading, error, getById, ensureLoaded, sortedNews, reload } = useNewsData();
ensureLoaded();

const { toast } = useAppToast();
watch(
  error,
  (val) => {
    if (!val) return;
    toast.add({
      title:       'Не удалось загрузить новость',
      description: String(val),
      color:       'error',
      icon:        'i-lucide-alert-circle',
    });
  },
  { immediate: true },
);

const newsId = computed(() => String(route.params.id ?? '').trim());
const item   = computed(() => (newsId.value ? getById(newsId.value) : undefined));

const title    = computed(() => item.value?.title || (newsId.value ? `Новость #${newsId.value}` : 'Новость'));
const date     = computed(() => formatNewsDate(item.value?.date ?? null));
const imageSrc = computed(() => resolveNewsImageSrc(item.value?.imagePath ?? null));

const isKiosk     = computed(() => route.matched?.some((r) => r.meta?.kiosk));
const newsListPath = computed(() => (isKiosk.value ? '/kiosk/news' : '/news'));
const isAdmin     = computed(() => currentRole.value === 'admin');

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
    prev: prev ? { title: prev.title || `Новость #${prev.id}`, description: prev.description.slice(0, 140), to: `${prefix}${prev.id}` } : null,
    next: next ? { title: next.title || `Новость #${next.id}`, description: next.description.slice(0, 140), to: `${prefix}${next.id}` } : null,
  };
});

// ─── Edit form ────────────────────────────────────────────────────────────────
const editOpen  = ref(false);
const formState = reactive({
  title:       '',
  category:    '',
  description: '',
  date:        '',
  image_path:  '',
});

type SingleDateValue = { value?: any } | any | null;
const formDateValue   = ref<SingleDateValue>(null);
const formImageFile   = ref<File | null>(null);
const formImagePreview = ref('');
const formSubmitting  = ref(false);
const formError       = ref<string | null>(null);

const categoryOptions = [
  { value: 'Новости',     label: 'Новости' },
  { value: 'Мероприятия', label: 'Мероприятия' },
  { value: 'Объявления',  label: 'Объявления' },
  { value: 'Архив',       label: 'Архив' },
];

watch(formDateValue, (val) => {
  const d = (val as any)?.value ?? val;
  formState.date = d && typeof d.toString === 'function' ? d.toString() : '';
}, { immediate: true });

const deleteConfirmOpen  = ref(false);
const deleteSubmitting   = ref(false);
const deleteError        = ref<string | null>(null);

const coverPreview = computed(() => {
  if (formImagePreview.value) return formImagePreview.value;
  return resolveNewsImageSrc(formState.image_path.trim() || null) ?? '';
});

function resetEditForm() {
  formState.title       = '';
  formState.category    = '';
  formState.description = '';
  formState.date        = '';
  formState.image_path  = '';
  formDateValue.value   = null;
  formImageFile.value   = null;
  if (formImagePreview.value) URL.revokeObjectURL(formImagePreview.value);
  formImagePreview.value = '';
  formError.value        = null;
}

function openEdit() {
  const n = item.value;
  if (!n) return;
  resetEditForm();
  formState.title       = n.title       ?? '';
  formState.category    = n.category    ?? '';
  formState.description = n.description ?? '';
  formState.image_path  = n.imagePath   ?? '';
  formDateValue.value   = n.date ? new Date(n.date) : new Date();
  editOpen.value = true;
}

function onFormImageSelected(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  formImageFile.value = file;
  if (formImagePreview.value) URL.revokeObjectURL(formImagePreview.value);
  formImagePreview.value = file ? URL.createObjectURL(file) : '';
}

async function uploadImage(file: File): Promise<string> {
  const fd  = new FormData();
  fd.append('image', file);
  const res  = await fetch('/api/Upload/upload.php', { method: 'POST', body: fd });
  const json = (await res.json()) as { success?: boolean; message?: string; data?: { image?: string } };
  if (!json.success) throw new Error(json.message || 'Ошибка загрузки изображения');
  const path = json.data?.image?.trim();
  if (!path) throw new Error('Сервер не вернул путь к изображению');
  return path;
}

async function handleEditSubmit() {
  const n = item.value;
  if (!n?.id) return;
  if (!formState.title.trim()) {
    formError.value = 'Укажите заголовок новости.';
    return;
  }
  if (!formState.date) {
    formError.value = 'Выберите дату.';
    return;
  }
  formSubmitting.value = true;
  formError.value      = null;
  try {
    let imagePath = formState.image_path.trim() || null;
    if (formImageFile.value) {
      imagePath = await uploadImage(formImageFile.value);
    }

    const res  = await fetch(`/api/news.php?id=${encodeURIComponent(n.id)}`, {
      method:  'PUT',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({
        title:       formState.title.trim(),
        category:    formState.category.trim(),
        description: formState.description.trim(),
        date:        formState.date,
        image_path:  imagePath,
      }),
    });
    const json = (await res.json()) as { success?: boolean; message?: string };
    if (!json.success) throw new Error(json.message || 'Ошибка сохранения');

    toast.add({ title: 'Новость сохранена', color: 'success', icon: 'i-lucide-check-circle' });
    editOpen.value = false;
    resetEditForm();
    await reload();
  } catch (e: unknown) {
    formError.value = e instanceof Error ? e.message : 'Ошибка запроса';
  } finally {
    formSubmitting.value = false;
  }
}

watch(deleteError, (val) => {
  if (!val) return;
  toast.add({ title: 'Не удалось удалить', description: String(val), color: 'error', icon: 'i-lucide-alert-circle' });
});

async function handleDelete() {
  const n = item.value;
  if (!n?.id) return;
  deleteSubmitting.value = true;
  deleteError.value      = null;
  try {
    const res  = await fetch(`/api/news.php?id=${encodeURIComponent(n.id)}`, { method: 'DELETE' });
    const json = (await res.json()) as { success?: boolean; message?: string };
    if (!json.success) throw new Error(json.message || 'Ошибка удаления');

    toast.add({ title: 'Новость удалена', color: 'success', icon: 'i-lucide-check-circle' });
    deleteConfirmOpen.value = false;
    void router.push(newsListPath.value);
  } catch (e: unknown) {
    deleteError.value = e instanceof Error ? e.message : 'Ошибка удаления';
  } finally {
    deleteSubmitting.value = false;
  }
}
</script>

<template>
  <UMain class="relative flex flex-col w-full h-full min-h-0 gap-0">
    <div class="flex-1 min-h-0 overflow-y-auto max-w-full w-full sm:p-0 md:p-0 lg:p-0 xl:p-0 scrollbar-hide mx-0 pb-8">
      <div v-if="loading" class="space-y-6">
        <USkeleton class="h-96 rounded-lg" />
        <USkeleton class="h-64 rounded-lg" />
      </div>

      <div v-else-if="item" class="flex flex-col gap-6 p-px">
        <!-- Hero -->
        <div class="relative overflow-hidden rounded-lg ring ring-default bg-default">
          <div class="relative h-96">
            <img
              v-if="imageSrc"
              :src="imageSrc"
              :alt="title"
              class="absolute inset-0 h-full w-full object-cover"
              loading="lazy"
            />
            <div
              class="absolute inset-0"
              :class="imageSrc
                ? 'bg-linear-to-t from-black/85 via-black/35 to-transparent'
                : 'bg-linear-to-br from-primary/12 via-transparent to-primary/8'"
            />
            <div class="absolute inset-0 p-5 sm:p-8 flex flex-col justify-end">
              <div class="flex flex-wrap items-center gap-2 mb-3">
                <UBadge v-if="item.category" color="primary" variant="solid" size="md">{{ item.category }}</UBadge>
                <UBadge v-if="date" color="neutral" variant="soft" size="md" class="backdrop-blur">
                  {{ date }}
                </UBadge>
              </div>
              <div class="flex flex-wrap items-end justify-between gap-4">
                <h1
                  class="text-2xl sm:text-4xl font-semibold tracking-tight min-w-0 flex-1"
                  :class="imageSrc ? 'text-white' : 'text-highlighted'"
                >
                  {{ title }}
                </h1>
                <div v-if="isAdmin" class="flex flex-wrap gap-2 shrink-0">
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
          <p class="text-default whitespace-pre-wrap">{{ item.description }}</p>
        </UCard>

        <UContentSurround v-if="surround.prev || surround.next" :prev="surround.prev" :next="surround.next" />
      </div>

      <div v-else class="py-10">
        <UEmpty
          icon="i-lucide-file-question"
          title="Новость не найдена"
          description="Возможно, она была удалена или ссылка неверная."
        />
      </div>
    </div>

    <!-- Edit slideover -->
    <USlideover v-model:open="editOpen" side="right" title="Редактирование новости" description="">
      <template #body>
        <UForm :state="formState" class="space-y-4" @submit.prevent="handleEditSubmit">
          <UFormField label="Название" name="title" required>
            <UInput v-model="formState.title" size="lg" class="w-full" placeholder="Заголовок новости" />
          </UFormField>

          <UFormField label="Категория" name="category">
            <USelect v-model="formState.category" :items="categoryOptions" placeholder="Выберите категорию" size="lg" class="w-full" />
          </UFormField>

          <UFormField label="Описание" name="description">
            <UTextarea v-model="formState.description" size="lg" class="w-full" :rows="6" placeholder="Текст новости..." />
          </UFormField>

          <UFormField label="Дата" name="date" required>
            <UInputDate v-model="formDateValue" size="lg" class="w-full">
              <template #trailing>
                <UPopover>
                  <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar" aria-label="Выбрать дату" class="px-0" />
                  <template #content>
                    <UCalendar v-model="formDateValue" class="p-2" />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </UFormField>

          <UFormField label="Обложка" name="image">
            <div class="flex flex-col gap-2 w-full">
              <input id="newsEditFileInput" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="onFormImageSelected" />
              <img v-if="coverPreview" :src="coverPreview" alt="Предпросмотр" class="w-full h-40 object-cover rounded-lg" />
              <label for="newsEditFileInput" class="flex items-center justify-center gap-2 cursor-pointer rounded-lg border border-default px-4 py-2.5 text-sm font-medium text-default hover:bg-elevated/50 transition-colors">
                <UIcon name="i-lucide-upload" class="text-base shrink-0" />
                {{ formImageFile ? 'Заменить изображение' : 'Загрузить изображение' }}
              </label>
              <p v-if="formImageFile" class="text-xs text-muted truncate px-1">{{ formImageFile.name }}</p>
            </div>
          </UFormField>

          <UAlert v-if="formError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="formError" />
        </UForm>
      </template>
      <template #footer>
        <div class="flex justify-between gap-3 items-center w-full">
          <UButton color="neutral" variant="outline" size="xl" class="w-full justify-center" @click="editOpen = false">Отмена</UButton>
          <UButton size="xl" class="w-full justify-center" :loading="formSubmitting" @click="handleEditSubmit">Сохранить</UButton>
        </div>
      </template>
    </USlideover>

    <!-- Delete confirm -->
    <UModal v-model:open="deleteConfirmOpen" title="Удалить новость?" description="">
      <template #body>
        <p class="text-default">
          Вы уверены, что хотите удалить <strong>«{{ item?.title }}»</strong>? Это действие нельзя отменить.
        </p>
        <UAlert v-if="deleteError" color="error" variant="subtle" icon="i-lucide-alert-circle" :description="deleteError" class="mt-3" />
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