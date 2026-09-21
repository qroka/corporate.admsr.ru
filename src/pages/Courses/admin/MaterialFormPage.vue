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

onMounted(async () => {
  loading.value = true;
  try {
    await store.loadCourse(courseId.value);
    if (isEdit.value) {
      const topic = store.topics.value.find((t) => t.id === topicId.value);
      const m = topic?.materials?.find((x) => x.id === materialId.value);
      if (m) {
        form.type = normalizeMaterialType(m.type);
        form.title = m.title;
        form.description = m.description || '';
        form.contentHtml = m.contentHtml || '';
        form.externalUrl = m.externalUrl || '';
        form.isRequired = m.isRequired !== false;
        form.minimumActiveSeconds = m.minimumActiveSeconds ?? 0;
      }
    }
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

async function onSave() {
  if (!form.title.trim()) {
    toast.add({ title: 'Укажите название', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  saving.value = true;
  try {
    if (isEdit.value && materialId.value) {
      await store.updateMaterial({
        materialId: materialId.value,
        title: form.title.trim(),
        description: form.description,
        contentHtml: form.contentHtml,
        externalUrl: form.externalUrl || null,
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
        externalUrl: form.externalUrl || null,
        isRequired: form.isRequired,
        minimumActiveSeconds: form.minimumActiveSeconds,
      });
    }
    toast.add({
      title: 'Материал сохранён',
      description: guideMode.value && !isEdit.value ? 'Шаг 3: настройте тест темы или вернитесь в курс' : undefined,
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
  <UMain class="flex flex-1 flex-col w-full max-w-3xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <h1 class="text-2xl font-medium text-highlighted">
      {{ isEdit ? 'Редактирование материала' : 'Новый материал' }}
    </h1>

    <UAlert
      v-if="guideMode && !isEdit"
      color="primary"
      variant="subtle"
      icon="i-lucide-list-ordered"
      title="Шаг 2 из 3 — материал"
      description="Добавьте текст, файл или ссылку. Дальше откроется тест темы."
    />

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 5" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <div v-else class="flex flex-col gap-4 min-w-0">
      <UFormField label="Тип">
        <USelect
          v-model="form.type"
          :items="typeItems"
          size="lg"
          class="w-full"
          :disabled="isEdit"
        />
      </UFormField>

      <UFormField label="Название" required>
        <UInput v-model="form.title" size="lg" class="w-full" />
      </UFormField>

      <UFormField label="Краткое описание">
        <UTextarea v-model="form.description" :rows="2" class="w-full" />
      </UFormField>

      <UFormField v-if="needsRichText" label="Содержимое" name="content">
        <UEditor
          v-slot="{ editor }"
          v-model="form.contentHtml"
          content-type="html"
          :extensions="newsEditorExtensions"
          :handlers="newsEditorHandlers"
          :ui="newsEditorSlideoverUi"
          placeholder="Текст материала…"
          class="w-full min-h-56 rounded-lg border border-default overflow-hidden"
        >
          <UEditorEmojiMenu
            :editor="editor"
            :items="newsEditorEmojiMenuItems"
            :append-to="appendEditorEmojiTo"
          />
          <UEditorToolbar
            :editor="editor"
            :items="newsEditorToolbarItems"
            class="sticky top-0 z-10 border-b border-default bg-default/95 backdrop-blur-sm px-2 py-1.5 overflow-x-auto"
          />
        </UEditor>
      </UFormField>

      <UFormField v-if="needsUrl" label="URL">
        <UInput v-model="form.externalUrl" size="lg" class="w-full" placeholder="https://" />
      </UFormField>

      <UFormField v-if="needsFile" label="Файл" :required="!isEdit">
        <UFileUpload
          v-model="file"
          label="Перетащите файл сюда"
          description="Или нажмите, чтобы выбрать. PDF, документы, изображения, видео."
          icon="i-lucide-upload"
          class="w-full min-h-32"
          layout="list"
        />
      </UFormField>

      <UFormField label="Обязательный">
        <USwitch v-model="form.isRequired" label="Нужен для завершения темы" />
      </UFormField>

      <UFormField label="Минимум активного времени (сек)">
        <UInput v-model.number="form.minimumActiveSeconds" type="number" :min="0" size="lg" class="w-full" />
      </UFormField>

      <div class="flex gap-2">
        <UButton color="primary" size="lg" :loading="saving" icon="i-lucide-check" @click="onSave">
          Сохранить
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
    </div>
  </UMain>
</template>
