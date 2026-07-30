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
const topicId = computed(() => {
  const raw = route.params.topicId;
  return raw ? Number(raw) : null;
});
const isEdit = computed(() => topicId.value != null && !Number.isNaN(topicId.value));

const loading = ref(true);
const saving = ref(false);
const deleteOpen = ref(false);
const deleteTarget = ref<{ id: number; title: string } | null>(null);
const deleting = ref(false);
const form = reactive({
  title: '',
  description: '',
  isRequired: true,
  minimumActiveSeconds: 0,
});

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'courses', query: { tab: 'manage' } } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: isEdit.value ? 'Тема' : 'Новая тема' },
]);

const materialTypeLabels: Record<string, string> = {
  rich_text: 'Текст',
  file: 'Файл',
  pdf: 'Файл',
  image: 'Файл',
  video: 'Файл',
  link: 'Ссылка',
};

const topicMaterials = computed(
  () => store.topics.value.find((t) => t.id === topicId.value)?.materials || [],
);

function materialTypeLabel(type?: string | null) {
  if (!type) return 'Материал';
  return materialTypeLabels[type] || type;
}

function askDeleteMaterial(m: { id: number; title?: string }) {
  deleteTarget.value = { id: m.id, title: m.title || 'Материал' };
  deleteOpen.value = true;
}

async function confirmDeleteMaterial() {
  if (!deleteTarget.value) return;
  deleting.value = true;
  try {
    await store.deleteMaterial(deleteTarget.value.id);
    toast.add({ title: 'Материал удалён', color: 'success', icon: 'i-lucide-check' });
    deleteOpen.value = false;
    deleteTarget.value = null;
    await store.loadCourse(courseId.value);
  } catch (e: any) {
    toast.add({ title: 'Не удалось удалить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    deleting.value = false;
  }
}

onMounted(async () => {
  loading.value = true;
  try {
    await store.loadCourse(courseId.value);
    if (isEdit.value) {
      const t = store.topics.value.find((x) => x.id === topicId.value);
      if (t) {
        form.title = t.title;
        form.description = t.description || '';
        form.isRequired = t.isRequired !== false;
        form.minimumActiveSeconds = t.minimumActiveSeconds ?? 0;
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
    toast.add({ title: 'Укажите название темы', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  saving.value = true;
  try {
    if (isEdit.value && topicId.value) {
      await store.updateTopic({
        topicId: topicId.value,
        title: form.title.trim(),
        description: form.description,
        isRequired: form.isRequired,
        minimumActiveSeconds: form.minimumActiveSeconds,
      });
      toast.add({ title: 'Тема сохранена', color: 'success', icon: 'i-lucide-check' });
    } else {
      const res = await store.createTopic({
        courseId: courseId.value,
        versionId: store.version.value?.id,
        title: form.title.trim(),
        description: form.description,
        isRequired: form.isRequired,
        minimumActiveSeconds: form.minimumActiveSeconds,
      }) as any;
      const newId = res?.topic?.id ?? res?.id;
      toast.add({ title: 'Тема создана', color: 'success', icon: 'i-lucide-check' });
      if (newId) {
        await router.replace({
          name: 'admin-course-topic-edit',
          params: { courseId: courseId.value, topicId: newId },
        });
        return;
      }
    }
    await router.push({ name: 'admin-course-workspace', params: { courseId: courseId.value } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось сохранить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full max-w-3xl mx-auto min-w-0 h-full min-h-0 gap-4 overflow-x-hidden">
    <UBreadcrumb :items="crumbs" />
    <div class="flex items-center justify-between gap-3 flex-wrap min-w-0">
      <h1 class="text-2xl font-medium text-highlighted break-words">
        {{ isEdit ? 'Редактирование темы' : 'Новая тема' }}
      </h1>
    </div>

    <div v-if="loading" class="flex flex-col gap-3 p-1">
      <USkeleton v-for="n in 4" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <div v-else class="flex flex-col gap-4 min-w-0 p-1">
      <UFormField label="Название" required>
        <UInput v-model="form.title" size="lg" class="w-full" />
      </UFormField>
      <UFormField label="Описание">
        <UTextarea v-model="form.description" :rows="4" class="w-full" />
      </UFormField>
      <UFormField label="Обязательная тема">
        <USwitch v-model="form.isRequired" label="Нужна для завершения курса" />
      </UFormField>
      <UFormField label="Минимум активного времени (сек)">
        <UInput v-model.number="form.minimumActiveSeconds" type="number" :min="0" size="lg" class="w-full" />
      </UFormField>

      <section v-if="isEdit" class="flex flex-col gap-2 pt-2 min-w-0">
        <div class="flex items-center justify-between gap-3 flex-wrap">
          <h2 class="text-lg font-medium">Материалы</h2>
          <UButton
            v-if="topicId"
            color="primary"
            variant="soft"
            size="sm"
            icon="i-lucide-plus"
            :to="{ name: 'admin-course-material-create', params: { courseId, topicId } }"
          >
            Добавить материал
          </UButton>
        </div>
        <ul
          v-if="topicMaterials.length"
          class="flex flex-col gap-2 list-none p-0 m-0 min-w-0"
        >
          <li
            v-for="m in topicMaterials"
            :key="m.id"
            class="rounded-lg ring-1 ring-default p-3 flex items-center justify-between gap-2 min-w-0"
          >
            <div class="min-w-0">
              <p class="font-medium break-words">{{ m.title }}</p>
              <p class="text-xs text-dimmed">{{ materialTypeLabel(m.type) }}</p>
            </div>
            <div class="flex items-center gap-1 shrink-0">
              <UButton
                color="neutral"
                variant="soft"
                size="sm"
                icon="i-lucide-pencil"
                :to="{ name: 'admin-course-material-edit', params: { courseId, topicId, materialId: m.id } }"
              >
                Изменить
              </UButton>
              <UButton
                color="error"
                variant="ghost"
                size="sm"
                icon="i-lucide-trash-2"
                square
                aria-label="Удалить материал"
                @click="askDeleteMaterial(m)"
              />
            </div>
          </li>
        </ul>
        <UEmpty
          v-else
          icon="i-lucide-file"
          title="Нет материалов"
          description="Добавьте текст, файл или ссылку."
          class="py-6"
        >
          <template #actions>
            <UButton
              v-if="topicId"
              color="primary"
              icon="i-lucide-plus"
              :to="{ name: 'admin-course-material-create', params: { courseId, topicId } }"
            >
              Добавить материал
            </UButton>
          </template>
        </UEmpty>
      </section>

      <div class="flex gap-2">
        <UButton color="primary" size="lg" :loading="saving" icon="i-lucide-check" @click="onSave">
          Сохранить
        </UButton>
        <UButton color="neutral" variant="ghost" size="lg" :to="{ name: 'admin-course-workspace', params: { courseId } }">
          Назад
        </UButton>
      </div>
    </div>

    <UModal
      v-model:open="deleteOpen"
      title="Удалить материал?"
      description="Материал будет удалён из темы. Это действие нельзя отменить."
    >
      <template #body>
        <p class="text-sm text-muted">
          Материал:
          <span class="text-highlighted font-medium">{{ deleteTarget?.title || '—' }}</span>
        </p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="deleteOpen = false">Отмена</UButton>
          <UButton color="error" icon="i-lucide-trash-2" :loading="deleting" @click="confirmDeleteMaterial">
            Удалить
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
