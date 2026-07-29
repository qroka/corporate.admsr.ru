<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { BreadcrumbItem } from '@nuxt/ui';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useUsersData } from '../../../composables/useUsersData';
import { useAppToast } from '../../../composables/useAppToast';
import OfoMultiSelect from '../../../components/OfoMultiSelect.vue';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const { users, ensureLoaded: ensureUsers } = useUsersData();

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const previewing = ref(false);
const assigning = ref(false);

const selectedUsers = ref<number[]>([]);
const ofoIds = ref<number[]>([]);
const includeChildren = ref(true);
const startsAt = ref('');
const deadlineAt = ref('');
const preview = ref<{ recipients?: any[]; count?: number; skipped?: number } | null>(null);

const userItems = computed(() =>
  users.value
    .map((u) => ({ label: u.fullName || String(u.id), value: u.id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru')),
);

const crumbs = computed<BreadcrumbItem[]>(() => [
  { label: 'Курсы', to: { name: 'admin-courses' } },
  { label: store.current.value?.title || 'Курс', to: { name: 'admin-course-workspace', params: { courseId: courseId.value } } },
  { label: 'Назначение' },
]);

const canAssign = computed(
  () => (selectedUsers.value.length > 0 || ofoIds.value.length > 0) && store.version.value?.status === 'published',
);

onMounted(async () => {
  ensureUsers();
  try {
    await store.loadCourse(courseId.value);
  } catch (e: any) {
    toast.add({ title: 'Ошибка', description: e?.message, color: 'error', icon: 'i-lucide-alert-circle' });
  } finally {
    loading.value = false;
  }
});

function payload() {
  return {
    courseId: courseId.value,
    versionId: store.version.value?.id,
    userIds: selectedUsers.value,
    ofoIds: ofoIds.value,
    includeChildren: includeChildren.value,
    startsAt: startsAt.value || null,
    deadlineAt: deadlineAt.value || null,
  };
}

async function onPreview() {
  if (!canAssign.value) {
    toast.add({ title: 'Выберите сотрудников или ОФО', color: 'warning', icon: 'i-lucide-alert-triangle' });
    return;
  }
  previewing.value = true;
  try {
    preview.value = (await store.assignPreview(payload())) as any;
  } catch (e: any) {
    toast.add({ title: 'Не удалось сформировать список', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    previewing.value = false;
  }
}

async function onAssign() {
  if (!canAssign.value) return;
  assigning.value = true;
  try {
    const res = (await store.assign(payload())) as any;
    toast.add({
      title: 'Курс назначен',
      description: `Создано записей: ${res?.createdEnrollments ?? res?.enrollments ?? '—'}`,
      color: 'success',
      icon: 'i-lucide-check',
    });
    await router.push({ name: 'admin-course-results', params: { courseId: courseId.value } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось назначить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    assigning.value = false;
  }
}

const previewList = computed(() => {
  const p = preview.value as any;
  if (!p) return [];
  return p.recipients || p.users || p.items || [];
});
</script>

<template>
  <UMain class="flex flex-1 flex-col w-full h-full min-h-0 gap-4 max-w-3xl">
    <UBreadcrumb :items="crumbs" />
    <h1 class="text-2xl font-medium text-highlighted">Назначение курса</h1>

    <UAlert
      v-if="!loading && store.version.value?.status !== 'published'"
      color="warning"
      variant="subtle"
      icon="i-lucide-alert-triangle"
      title="Курс ещё не опубликован"
      description="Назначать можно только опубликованную версию."
    />

    <div v-if="loading" class="flex flex-col gap-3">
      <USkeleton v-for="n in 5" :key="n" class="h-12 w-full rounded-lg" />
    </div>

    <div v-else class="flex flex-col gap-5">
      <UFormField label="Сотрудники">
          <USelectMenu
            v-model="selectedUsers"
            :items="userItems"
            multiple
            searchable
            value-key="value"
            label-key="label"
            placeholder="Выберите сотрудников"
            size="lg"
            class="w-full"
            :content="{ align: 'start', sideOffset: 8 }"
          />
      </UFormField>

      <UFormField label="ОФО">
        <OfoMultiSelect v-model="ofoIds" />
      </UFormField>

      <UFormField>
        <UCheckbox v-model="includeChildren" label="Включать дочерние ОФО" />
      </UFormField>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <UFormField label="Дата начала">
          <UInput v-model="startsAt" type="datetime-local" size="lg" class="w-full" />
        </UFormField>
        <UFormField label="Дедлайн">
          <UInput v-model="deadlineAt" type="datetime-local" size="lg" class="w-full" />
        </UFormField>
      </div>

      <div class="flex flex-wrap gap-2">
        <UButton
          color="neutral"
          variant="soft"
          size="lg"
          icon="i-lucide-eye"
          :loading="previewing"
          :disabled="!canAssign"
          @click="onPreview"
        >
          Предпросмотр получателей
        </UButton>
        <UButton
          color="primary"
          size="lg"
          icon="i-lucide-user-plus"
          :loading="assigning"
          :disabled="!canAssign"
          @click="onAssign"
        >
          Назначить
        </UButton>
      </div>

      <section v-if="preview" class="flex flex-col gap-2">
        <h2 class="text-lg font-medium">
          Получатели
          <span class="text-sm font-normal text-dimmed">
            ({{ preview.count ?? previewList.length }})
          </span>
        </h2>
        <UEmpty
          v-if="!previewList.length"
          icon="i-lucide-users"
          title="Список пуст"
          description="Проверьте выбор ОФО и сотрудников."
          class="py-6"
        />
        <ul v-else class="max-h-64 overflow-y-auto flex flex-col gap-1 list-none p-0 m-0 rounded-xl ring-1 ring-default divide-y divide-default">
          <li
            v-for="(r, i) in previewList"
            :key="r.id ?? r.userId ?? i"
            class="px-3 py-2 text-sm"
          >
            {{ r.fio || r.fullName || r.name || r.login || ('Сотрудник ' + (r.userId || r.id)) }}
          </li>
        </ul>
      </section>
    </div>
  </UMain>
</template>
