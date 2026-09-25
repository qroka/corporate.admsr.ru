<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { z } from 'zod';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';
import { activeTimeHint } from '../courseDuration';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
useAdminCoursePortalBreadcrumbs();

const courseId = computed(() => Number(route.params.courseId));
const topicId = computed(() => {
  const raw = route.params.topicId;
  return raw ? Number(raw) : null;
});
const isEdit = computed(() => topicId.value != null && !Number.isNaN(topicId.value));
const guideMode = computed(() => String(route.query.guide || '') === '1');

const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const deleteOpen = ref(false);
const deleteTarget = ref<{ id: number; title: string } | null>(null);
const deleting = ref(false);

const schema = z.object({
  title: z.string().trim().min(1, 'Укажите название темы'),
  description: z.string(),
  minimumActiveSeconds: z
    .number({ message: 'Введите число секунд' })
    .int('Только целые секунды')
    .min(0, 'Не может быть отрицательным')
    .max(86400, 'Не больше суток'),
});

const form = reactive({
  title: '',
  description: '',
  isRequired: true,
  minimumActiveSeconds: 0,
});

/** Поле остаётся в секундах, как в API, но рядом показываем то же число по-человечески. */
const minimumActiveHint = computed(() => activeTimeHint(form.minimumActiveSeconds, 'тему'));

const materialTypeLabels: Record<string, string> = {
  rich_text: 'Текст',
  file: 'Файл',
  pdf: 'Файл',
  image: 'Файл',
  video: 'Файл',
  link: 'Ссылка',
};

const currentTopic = computed(() => store.topics.value.find((t) => t.id === topicId.value) || null);
const topicMaterials = computed(() => currentTopic.value?.materials || []);
/** Правим тему, которой нет в этой версии курса — форма бесполезна. */
const topicMissing = computed(() => isEdit.value && !loading.value && !loadError.value && !currentTopic.value);

const submitLabel = computed(() => {
  if (isEdit.value) return 'Сохранить';
  return guideMode.value ? 'Сохранить и добавить материал' : 'Создать тему';
});

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

async function load() {
  loading.value = true;
  loadError.value = null;
  try {
    await store.loadCourse(courseId.value);
    const t = currentTopic.value;
    if (isEdit.value && t) {
      form.title = t.title;
      form.description = t.description || '';
      form.isRequired = t.isRequired !== false;
      form.minimumActiveSeconds = t.minimumActiveSeconds ?? 0;
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
      const versionId = store.version.value?.id;
      if (!versionId) throw new Error('Версия курса не загружена — обновите страницу');
      const res = (await store.createTopic({
        courseId: courseId.value,
        versionId,
        title: form.title.trim(),
        description: form.description,
        isRequired: form.isRequired,
        minimumActiveSeconds: form.minimumActiveSeconds,
      })) as any;
      const newId = res?.topic?.id ?? res?.id;
      toast.add({
        title: 'Тема создана',
        description: guideMode.value ? 'Шаг 3 из 4: добавьте материал' : undefined,
        color: 'success',
        icon: 'i-lucide-check',
      });
      if (newId) {
        if (guideMode.value) {
          await router.push({
            name: 'admin-course-material-create',
            params: { courseId: courseId.value, topicId: newId },
            query: { guide: '1' },
          });
          return;
        }
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
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        :title="isEdit ? 'Редактирование темы' : 'Новая тема'"
        :description="isEdit
          ? 'Название, описание и материалы темы'
          : 'Тема — это раздел курса, внутри которого лежат материалы'"
      />

      <UAlert
        v-if="guideMode && !isEdit"
        color="primary"
        variant="subtle"
        icon="i-lucide-list-ordered"
        title="Шаг 2 из 4 — тема"
        description="Укажите название. После сохранения сразу откроется добавление материала."
      />

      <div v-if="loading" class="flex flex-col gap-4">
        <USkeleton class="h-16 w-full rounded-lg" />
        <USkeleton class="h-28 w-full rounded-lg" />
        <USkeleton class="h-12 w-2/3 rounded-lg" />
        <USkeleton class="h-16 w-full rounded-lg" />
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
        v-else-if="topicMissing"
        color="warning"
        variant="subtle"
        icon="i-lucide-file-question"
        title="Тема не найдена"
        description="Возможно, её удалили или она относится к другой версии курса."
      >
        <template #actions>
          <UButton color="neutral" variant="soft" :to="{ name: 'admin-course-workspace', params: { courseId } }">
            Вернуться к курсу
          </UButton>
        </template>
      </UAlert>

      <template v-else>
        <UForm :schema="schema" :state="form" class="flex flex-col gap-4 min-w-0" @submit="onSubmit">
          <UFormField label="Название" name="title" required>
            <UInput
              v-model="form.title"
              size="lg"
              class="w-full"
              :autofocus="!isEdit"
              placeholder="Например: Пожарная безопасность на рабочем месте"
            />
          </UFormField>

          <UFormField label="Описание" name="description" hint="Необязательно">
            <UTextarea
              v-model="form.description"
              :rows="4"
              class="w-full"
              placeholder="Что сотрудник узнает из этой темы"
            />
          </UFormField>

          <UFormField label="Обязательная тема" name="isRequired">
            <USwitch v-model="form.isRequired" label="Нужна для завершения курса" />
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
              :to="{ name: 'admin-course-workspace', params: { courseId } }"
            >
              Назад
            </UButton>
          </div>
        </UForm>

        <section v-if="isEdit" class="flex flex-col gap-3 min-w-0 border-t border-accented pt-5">
          <div class="flex items-center justify-between gap-3 flex-wrap">
            <h2 class="text-lg font-medium text-highlighted">Материалы</h2>
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

          <ul v-if="topicMaterials.length" class="flex flex-col gap-2 list-none p-0 m-0 min-w-0">
            <li
              v-for="m in topicMaterials"
              :key="m.id"
              class="rounded-lg ring-1 ring-accented p-3 flex items-center justify-between gap-2 min-w-0"
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
            description="Добавьте текст, файл или ссылку — без материалов тему нельзя пройти."
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
      </template>
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
