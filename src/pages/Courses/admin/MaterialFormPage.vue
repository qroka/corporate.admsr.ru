<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();

const courseId = computed(() => Number(route.params.courseId));
const topicId = computed(() => Number(route.params.topicId));
const materialId = computed(() => {
  const raw = route.params.materialId;
  return raw ? Number(raw) : null;
});
const isEdit = computed(() => materialId.value != null);

const typeItems = [
  { label: 'Текст (HTML)', value: 'rich_text' },
  { label: 'Файл', value: 'file' },
  { label: 'PDF', value: 'pdf' },
  { label: 'Изображение', value: 'image' },
  { label: 'Видео', value: 'video' },
  { label: 'Ссылка', value: 'link' },
];

const loading = ref(true);
const saving = ref(false);
const file = ref<File | null>(null);

const form = reactive({
  type: 'rich_text' as string,
  title: '',
  description: '',
  contentHtml: '',
  externalUrl: '',
  isRequired: true,
  minimumActiveSeconds: 0,
});

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Тема', to: { name: 'admin-course-topic-edit', params: { courseId: courseId.value, topicId: topicId.value } } },
  { label: isEdit.value ? 'Материал' : 'Новый материал' },
]);

const needsFile = computed(() => ['file', 'pdf', 'image', 'video'].includes(form.type));
const needsUrl = computed(() => form.type === 'link');
const needsHtml = computed(() => form.type === 'rich_text');

onMounted(async () => {
  loading.value = true;
  try {
    await store.loadCourse(courseId.value);
    if (isEdit.value) {
      const topic = store.topics.value.find((t) => t.id === topicId.value);
      const m = topic?.materials?.find((x) => x.id === materialId.value);
      if (m) {
        form.type = m.type;
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

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement;
  file.value = input.files?.[0] || null;
  if (file.value && !form.title) form.title = file.value.name;
}

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
    toast.add({ title: 'Материал сохранён', color: 'success', icon: 'i-lucide-check' });
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
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">
      {{ isEdit ? 'Редактирование материала' : 'Новый материал' }}
    </h1>

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 5" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <div v-else class="flex flex-col gap-4">
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

      <UFormField label="Описание">
        <UTextarea v-model="form.description" :rows="2" class="w-full" />
      </UFormField>

      <UFormField v-if="needsHtml" label="Содержимое (HTML)">
        <UTextarea v-model="form.contentHtml" :rows="10" class="w-full font-mono text-sm" placeholder="<p>Текст…</p>" />
      </UFormField>

      <UFormField v-if="needsUrl" label="URL">
        <UInput v-model="form.externalUrl" size="lg" class="w-full" placeholder="https://" />
      </UFormField>

      <UFormField v-if="needsFile" label="Файл">
        <input
          type="file"
          class="block w-full text-sm text-muted file:mr-3 file:rounded-lg file:border-0 file:bg-primary file:px-3 file:py-2 file:text-white"
          @change="onFileChange"
        >
        <p v-if="file" class="text-xs text-dimmed mt-1">{{ file.name }}</p>
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
