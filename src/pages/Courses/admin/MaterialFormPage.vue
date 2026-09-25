<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { newsEditorToolbarItems } from '../../../composables/newsEditorToolbar';
import { newsEditorExtensions, newsEditorEmojiMenuItems } from '../../../composables/newsEditorExtensions';
import { newsEditorHandlers } from '../../../composables/newsEditorHandlers';
import { newsEditorSlideoverUi } from '../../../composables/newsEditorSlideoverUi';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';
import { activeTimeHint } from '../courseDuration';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
useAdminCoursePortalBreadcrumbs();

const courseId = computed(() => Number(route.params.courseId));
const topicId = computed(() => Number(route.params.topicId));
const materialId = computed(() => {
  const raw = route.params.materialId;
  return raw ? Number(raw) : null;
});
const isEdit = computed(() => materialId.value != null);
const guideMode = computed(() => String(route.query.guide || '') === '1');

const typeItems = [
  { label: 'Текст', value: 'rich_text' },
  { label: 'Файл', value: 'file' },
  { label: 'Ссылка', value: 'link' },
];

const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const file = ref<File | null>(null);

/** Меню эмодзи поверх формы */
const appendEditorEmojiTo = () => document.body;

const form = reactive({
  type: 'rich_text' as string,
  title: '',
  description: '',
  contentHtml: '',
  externalUrl: '',
  isRequired: true,
  minimumActiveSeconds: 0,
});

watch(file, (f) => {
  if (f && !form.title.trim()) form.title = f.name;
});

const FILE_TYPES = new Set(['file', 'pdf', 'image', 'video']);

function normalizeMaterialType(type?: string | null) {
  const t = String(type || 'rich_text');
  if (FILE_TYPES.has(t)) return 'file';
  if (t === 'link' || t === 'rich_text') return t;
  return 'rich_text';
}

const needsFile = computed(() => form.type === 'file');
const needsUrl = computed(() => form.type === 'link');
const needsRichText = computed(() => form.type === 'rich_text');

const topic = computed(() => store.topics.value.find((t) => t.id === topicId.value) || null);
const currentMaterial = computed(
  () => topic.value?.materials?.find((x) => x.id === materialId.value) || null,
);
/** Правим материал, которого нет в этой теме — форма бесполезна. */
const materialMissing = computed(
  () => isEdit.value && !loading.value && !loadError.value && !currentMaterial.value,
);

const minimumActiveHint = computed(() => activeTimeHint(form.minimumActiveSeconds, 'материал'));

const submitLabel = computed(() => {
  if (isEdit.value) return 'Сохранить';
  return guideMode.value ? 'Сохранить и перейти к тесту' : 'Добавить материал';
});

function isHtmlEmpty(html: string) {
  const text = String(html || '')
    .replace(/<[^>]*>/g, '')
    .replace(/&nbsp;/g, ' ')
    .trim();
  return !text && !String(html || '').includes('<img');
}

/**
 * Требования зависят от типа: ссылке нужен URL, файлу — сам файл,
 * тексту — непустое содержимое. Раньше всё это молча сохранялось пустым.
 */
function validate(state: typeof form) {
  const errors: { name: string; message: string }[] = [];

  if (!state.title.trim()) {
    errors.push({ name: 'title', message: 'Укажите название материала' });
  }

  if (state.type === 'link') {
    const url = state.externalUrl.trim();
    if (!url) errors.push({ name: 'externalUrl', message: 'Укажите ссылку' });
    else if (!/^https?:\/\/\S+$/i.test(url)) {
      errors.push({ name: 'externalUrl', message: 'Ссылка должна начинаться с http:// или https://' });
    }
  }

  if (state.type === 'rich_text' && isHtmlEmpty(state.contentHtml)) {
    errors.push({ name: 'contentHtml', message: 'Добавьте текст материала' });
  }

  if (state.type === 'file' && !isEdit.value && !file.value) {
    errors.push({ name: 'file', message: 'Выберите файл' });
  }

  const secs = Number(state.minimumActiveSeconds);
  if (!Number.isInteger(secs) || secs < 0) {
    errors.push({ name: 'minimumActiveSeconds', message: 'Целое число секунд, не меньше нуля' });
  } else if (secs > 86400) {
    errors.push({ name: 'minimumActiveSeconds', message: 'Не больше суток' });
  }

  return errors;
}

async function load() {
  loading.value = true;
  loadError.value = null;
  try {
    await store.loadCourse(courseId.value);
    const m = currentMaterial.value;
    if (isEdit.value && m) {
      form.type = normalizeMaterialType(m.type);
      form.title = m.title;
      form.description = m.description || '';
      form.contentHtml = m.contentHtml || '';
      form.externalUrl = m.externalUrl || '';
      form.isRequired = m.isRequired !== false;
      form.minimumActiveSeconds = m.minimumActiveSeconds ?? 0;
    }
  } catch (e: any) {
    loadError.value = e?.message || 'Не удалось загрузить курс';
  } finally {
    loading.value = false;
  }
}

onMounted(load);

async function onSubmit() {
  saving.value = true;
  try {
    if (isEdit.value && materialId.value) {
      await store.updateMaterial({
        materialId: materialId.value,
        title: form.title.trim(),
        description: form.description,
        contentHtml: form.contentHtml,
        externalUrl: form.externalUrl.trim() || null,
        isRequired: form.isRequired,
        minimumActiveSeconds: form.minimumActiveSeconds,
      });
      if (needsFile.value && file.value) {
        await store.uploadMaterialFile(topicId.value, file.value, {
          materialId: String(materialId.value),
          title: form.title.trim(),
          type: form.type,
        });
      }
    } else if (needsFile.value && file.value) {
      await store.uploadMaterialFile(topicId.value, file.value, {
        title: form.title.trim(),
        description: form.description,
        type: form.type,
        isRequired: form.isRequired ? '1' : '0',
        minimumActiveSeconds: String(form.minimumActiveSeconds || 0),
      });
    } else {
      await store.createMaterial({
        topicId: topicId.value,
        type: form.type,
        title: form.title.trim(),
        description: form.description,
        contentHtml: form.contentHtml || null,
        externalUrl: form.externalUrl.trim() || null,
        isRequired: form.isRequired,
        minimumActiveSeconds: form.minimumActiveSeconds,
      });
    }
    toast.add({
      title: 'Материал сохранён',
      description:
        guideMode.value && !isEdit.value ? 'Шаг 4 из 4: настройте тест темы или вернитесь в курс' : undefined,
      color: 'success',
      icon: 'i-lucide-check',
    });
    if (guideMode.value && !isEdit.value) {
      await router.push({
        name: 'admin-course-topic-test',
        params: { courseId: courseId.value, topicId: topicId.value },
        query: { guide: '1' },
      });
      return;
    }
    await router.push({
      name: 'admin-course-topic-edit',
      params: { courseId: courseId.value, topicId: topicId.value },
    });
  } catch (e: any) {
    toast.add({ title: 'Не удалось сохранить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        :title="isEdit ? 'Редактирование материала' : 'Новый материал'"
        description="Текст, файл или ссылка — то, что сотрудник изучает внутри темы"
      />

      <UAlert
        v-if="guideMode && !isEdit"
        color="primary"
        variant="subtle"
        icon="i-lucide-list-ordered"
        title="Шаг 3 из 4 — материал"
        description="Добавьте текст, файл или ссылку. Дальше откроется тест темы."
      />

      <div v-if="loading" class="flex flex-col gap-4">
        <USkeleton class="h-16 w-full rounded-lg" />
        <USkeleton class="h-16 w-full rounded-lg" />
        <USkeleton class="h-56 w-full rounded-lg" />
        <USkeleton class="h-12 w-2/3 rounded-lg" />
      </div>

      <UAlert
        v-else-if="loadError"
        color="warning"
        variant="subtle"
        icon="i-lucide-server"
        title="Не удалось загрузить курс"
        :description="loadError"
      >
        <template #actions>
          <UButton color="warning" icon="i-lucide-rotate-ccw" @click="load">Повторить</UButton>
          <UButton color="neutral" variant="ghost" :to="{ name: 'admin-courses' }">К списку</UButton>
        </template>
      </UAlert>

      <UAlert
        v-else-if="materialMissing"
        color="warning"
        variant="subtle"
        icon="i-lucide-file-question"
        title="Материал не найден"
        description="Возможно, его удалили или он относится к другой теме."
      >
        <template #actions>
          <UButton
            color="neutral"
            variant="soft"
            :to="{ name: 'admin-course-topic-edit', params: { courseId, topicId } }"
          >
            Вернуться к теме
          </UButton>
        </template>
      </UAlert>

      <UForm
        v-else
        :state="form"
        :validate="validate"
        class="flex flex-col gap-4 min-w-0"
        @submit="onSubmit"
      >
        <UFormField
          label="Тип"
          name="type"
          :description="isEdit ? 'Тип материала нельзя изменить после создания.' : undefined"
        >
          <USelect v-model="form.type" :items="typeItems" size="lg" class="w-full" :disabled="isEdit" />
        </UFormField>

        <UFormField label="Название" name="title" required>
          <UInput
            v-model="form.title"
            size="lg"
            class="w-full"
            :autofocus="!isEdit"
            placeholder="Как материал называется для сотрудника"
          />
        </UFormField>

        <UFormField label="Краткое описание" name="description" hint="Необязательно">
          <UTextarea v-model="form.description" :rows="2" class="w-full" />
        </UFormField>

        <UFormField v-if="needsRichText" label="Содержимое" name="contentHtml" required>
          <UEditor
            v-slot="{ editor }"
            v-model="form.contentHtml"
            content-type="html"
            :extensions="newsEditorExtensions"
            :handlers="newsEditorHandlers"
            :ui="newsEditorSlideoverUi"
            placeholder="Текст материала…"
            class="w-full min-h-56 rounded-lg border border-accented overflow-hidden"
          >
            <UEditorEmojiMenu
              :editor="editor"
              :items="newsEditorEmojiMenuItems"
              :append-to="appendEditorEmojiTo"
            />
            <UEditorToolbar
              :editor="editor"
              :items="newsEditorToolbarItems"
              class="sticky top-0 z-10 border-b border-accented bg-default/95 backdrop-blur-sm px-2 py-1.5 overflow-x-auto"
            />
          </UEditor>
        </UFormField>

        <UFormField v-if="needsUrl" label="Ссылка" name="externalUrl" required>
          <UInput v-model="form.externalUrl" size="lg" class="w-full" placeholder="https://" />
        </UFormField>

        <UFormField
          v-if="needsFile"
          label="Файл"
          name="file"
          :required="!isEdit"
          :description="isEdit ? 'Загрузите новый файл, чтобы заменить текущий.' : undefined"
        >
          <UFileUpload
            v-model="file"
            label="Перетащите файл сюда"
            description="Или нажмите, чтобы выбрать. PDF, документы, изображения, видео."
            icon="i-lucide-upload"
            class="w-full min-h-32"
            layout="list"
          />
        </UFormField>

        <UFormField label="Обязательный" name="isRequired">
          <USwitch v-model="form.isRequired" label="Нужен для завершения темы" />
        </UFormField>

        <UFormField
          label="Минимум активного времени"
          name="minimumActiveSeconds"
          hint="В секундах"
          :description="minimumActiveHint"
        >
          <UInput
            v-model.number="form.minimumActiveSeconds"
            type="number"
            :min="0"
            :step="30"
            size="lg"
            class="w-full"
          />
        </UFormField>

        <div class="flex gap-2 pt-1">
          <UButton type="submit" color="primary" size="lg" :loading="saving" icon="i-lucide-check">
            {{ submitLabel }}
          </UButton>
          <UButton
            color="neutral"
            variant="ghost"
            size="lg"
            :to="{ name: 'admin-course-topic-edit', params: { courseId, topicId } }"
          >
            Назад
          </UButton>
        </div>
      </UForm>
    </div>
  </UMain>
</template>
