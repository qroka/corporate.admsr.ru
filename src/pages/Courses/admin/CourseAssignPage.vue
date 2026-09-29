<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useUsersData } from '../../../composables/useUsersData';
import { useAppToast } from '../../../composables/useAppToast';
import { useOfoTree } from '../../../composables/useOfoTree';
import { useAdminCoursePortalBreadcrumbs } from '../useAdminCoursePortalBreadcrumbs';
import { formatDate } from '../courseDeadline';
import { plural } from '../courseDuration';

const route = useRoute();
const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
useAdminCoursePortalBreadcrumbs();
const { users, ensureLoaded: ensureUsers } = useUsersData();
const { categories, ensureLoaded: ensureOfo, rootUnitsOf } = useOfoTree();

const courseId = computed(() => Number(route.params.courseId));
const loading = ref(true);
const loadError = ref<string | null>(null);
const previewing = ref(false);
const assigning = ref(false);
const confirmOpen = ref(false);

const selectedUsers = ref<number[]>([]);
const ofoIds = ref<number[]>([]);
const includeChildren = ref(true);
/** `datetime-local`: «2026-10-01T09:00», локальное время браузера. */
const startsAt = ref('');
/** `date`: «2026-10-15» — последний день, когда курс ещё не просрочен. */
const deadlineDate = ref('');
const preview = ref<{ recipients?: any[]; count?: number } | null>(null);

const title = computed(() => store.current.value?.title || 'Обучение');
const isPublished = computed(() => store.version.value?.status === 'published');
const defaultDays = computed(() => {
  const n = Number(store.version.value?.defaultDeadlineDays || 0);
  return Number.isFinite(n) && n > 0 ? n : 0;
});

const userItems = computed(() =>
  users.value
    .map((u) => ({ label: u.fullName || String(u.id), value: u.id }))
    .sort((a, b) => a.label.localeCompare(b.label, 'ru')),
);

/** Значение пункта «Все сотрудники» в списке ОФО (id подразделений положительные). */
const ALL_STAFF = 0;

/** «Все сотрудники» + корневые ОФО — как в фильтре результатов (только названия, без категорий). */
const ofoItems = computed(() => {
  const items: { label: string; value: number }[] = [];
  const cats = [...categories.value].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id);
  for (const cat of cats) {
    for (const u of rootUnitsOf(cat.id)) items.push({ label: u.name, value: u.id });
  }
  items.sort((a, b) => a.label.localeCompare(b.label, 'ru'));
  return [[{ label: 'Все сотрудники', value: ALL_STAFF, icon: 'i-lucide-users' }], items];
});

const allStaff = computed(() => ofoIds.value.includes(ALL_STAFF));
const unitIds = computed(() => ofoIds.value.filter((id) => id !== ALL_STAFF));

/** «Все сотрудники» и конкретные ОФО взаимоисключающие: последний выбор побеждает. */
watch(ofoIds, (next, prev) => {
  const added = next.filter((id) => !prev.includes(id));
  if (added.includes(ALL_STAFF) && next.length > 1) ofoIds.value = [ALL_STAFF];
  else if (next.includes(ALL_STAFF) && added.length) ofoIds.value = next.filter((id) => id !== ALL_STAFF);
});

const ofoDescription = computed(() =>
  allStaff.value
    ? 'Курс получат все сотрудники, в том числе ещё не входившие на портал. Кто войдёт впервые позже — получит курс при входе.'
    : 'Кто придёт в выбранное подразделение позже (или впервые войдёт и выберет его), получит курс при входе.',
);

const hasRecipients = computed(() => selectedUsers.value.length > 0 || ofoIds.value.length > 0);

function todayIso() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/** Ошибка у поля срока: срок не в прошлом и не раньше начала. */
const deadlineError = computed(() => {
  if (!deadlineDate.value) return '';
  if (deadlineDate.value < todayIso()) return 'Срок не может быть в прошлом';
  if (startsAt.value && deadlineDate.value < startsAt.value.slice(0, 10)) return 'Срок раньше даты начала';
  return '';
});

const deadlineHint = computed(() => {
  if (deadlineDate.value) return 'Курс считается просроченным со следующего дня.';
  if (defaultDays.value) {
    return `Если не указать — ${defaultDays.value} ${plural(defaultDays.value, ['день', 'дня', 'дней'])} с момента, когда сотрудник получил курс (настройка курса).`;
  }
  return 'Необязательно. Без срока курс не станет просроченным.';
});

const canSubmit = computed(() => isPublished.value && hasRecipients.value && !deadlineError.value);

/**
 * Go разбирает даты как RFC3339: строку из `datetime-local` он не распознаёт
 * и отсчитывает срок от текущего момента. Поэтому отдаём ISO с часовым поясом.
 */
function toIsoOrNull(local: string, endOfDay = false) {
  if (!local) return null;
  const d = endOfDay ? new Date(`${local}T23:59:59`) : new Date(local);
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

function payload() {
  return {
    courseId: courseId.value,
    versionId: store.version.value?.id,
    userIds: selectedUsers.value,
    ofoIds: unitIds.value,
    allUsers: allStaff.value,
    includeChildren: includeChildren.value,
    startsAt: toIsoOrNull(startsAt.value),
    // null → сервер применит срок по умолчанию из версии, если он задан
    deadlineAt: toIsoOrNull(deadlineDate.value, true),
  };
}

async function load() {
  loading.value = true;
  loadError.value = null;
  ensureUsers();
  try {
    await Promise.all([ensureOfo(), store.loadCourse(courseId.value)]);
  } catch (e: any) {
    loadError.value = e?.message || 'Не удалось загрузить курс';
  } finally {
    loading.value = false;
  }
}

onMounted(load);

async function runPreview() {
  previewing.value = true;
  try {
    preview.value = (await store.assignPreview(payload())) as any;
    return true;
  } catch (e: any) {
    toast.add({ title: 'Не удалось сформировать список', description: e?.message, color: 'error', icon: 'i-lucide-x' });
    return false;
  } finally {
    previewing.value = false;
  }
}

/** «Назначить» сначала показывает, сколько людей получат курс, — назначение массовое. */
async function askAssign() {
  if (!canSubmit.value) return;
  if (await runPreview()) confirmOpen.value = true;
}

async function onAssign() {
  if (!canSubmit.value) return;
  assigning.value = true;
  try {
    const res = (await store.assign(payload())) as any;
    const created = Number(res?.enrollmentsCreated ?? 0);
    const skipped = Number(res?.skipped ?? 0);
    toast.add({
      title: created ? 'Курс назначен' : 'Новых назначений нет',
      description: [
        `Назначено: ${created}`,
        skipped ? `уже были назначены: ${skipped}` : '',
      ].filter(Boolean).join(', ') + '.',
      color: created ? 'success' : 'warning',
      icon: created ? 'i-lucide-check' : 'i-lucide-info',
    });
    confirmOpen.value = false;
    if (created) await router.push({ name: 'admin-course-results', params: { courseId: courseId.value } });
  } catch (e: any) {
    toast.add({ title: 'Не удалось назначить', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    assigning.value = false;
  }
}

const previewList = computed<any[]>(() => {
  const p = preview.value as any;
  if (!p) return [];
  return p.recipients || p.users || p.items || [];
});
const previewCount = computed(() => Number(preview.value?.count ?? previewList.value.length));
/** Сколько получателей ещё не выбрали ОФО — как правило, это те, кто не входил. */
const previewWithoutOfo = computed(() => Number((preview.value as any)?.withoutOfo ?? 0));

function recipientName(r: any) {
  return r.fio || r.fullName || r.name || r.login || `Сотрудник ${r.userId || r.id}`;
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8 *:shrink-0">
      <UPageHeader headline="Обучение" title="Назначение" :description="loading || loadError ? undefined : title" />

      <div v-if="loading" class="flex flex-col gap-4" aria-busy="true" aria-label="Загрузка">
        <USkeleton v-for="n in 4" :key="n" class="h-16 w-full rounded-lg" />
        <USkeleton class="h-10 w-64 rounded-lg" />
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
          <UButton color="neutral" variant="ghost" :to="{ name: 'admin-courses' }">К списку обучения</UButton>
        </template>
      </UAlert>

      <UAlert
        v-else-if="!isPublished"
        color="warning"
        variant="subtle"
        icon="i-lucide-alert-triangle"
        title="Курс ещё не опубликован"
        description="Назначать можно только опубликованный курс."
      >
        <template #actions>
          <UButton color="primary" icon="i-lucide-send" :to="{ name: 'admin-course-publish', params: { courseId } }">
            К публикации
          </UButton>
        </template>
      </UAlert>

      <div v-else class="flex flex-col gap-5">
        <section class="flex flex-col gap-4" aria-labelledby="assign-who-title">
          <h2 id="assign-who-title" class="text-lg font-bold leading-7 text-highlighted">Кому</h2>
          <UFormField label="Сотрудники" description="Выберите людей поимённо, подразделения целиком — или то и другое.">
            <USelectMenu
              v-model="selectedUsers"
              :items="userItems"
              multiple
              value-key="value"
              label-key="label"
              placeholder="Выберите сотрудников"
              size="lg"
              class="w-full"
              :search-input="{ placeholder: 'Поиск…' }"
              :content="{ align: 'start', sideOffset: 8 }"
            />
          </UFormField>

          <UFormField label="Подразделения (ОФО)" :description="ofoIds.length ? ofoDescription : undefined">
            <USelectMenu
              v-model="ofoIds"
              :items="ofoItems"
              multiple
              value-key="value"
              label-key="label"
              placeholder="Выберите ОФО или «Все сотрудники»"
              size="lg"
              color="neutral"
              class="w-full"
              :search-input="{ placeholder: 'Найти ОФО…' }"
              :content="{ align: 'start', sideOffset: 8 }"
            />
          </UFormField>

          <UFormField v-if="unitIds.length">
            <UCheckbox v-model="includeChildren" label="Включая вложенные подразделения" />
          </UFormField>
        </section>

        <section class="flex flex-col gap-4 border-t border-default pt-5" aria-labelledby="assign-when-title">
          <h2 id="assign-when-title" class="text-lg font-bold leading-7 text-highlighted">Когда</h2>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <UFormField label="Дата начала" hint="Необязательно" description="До этой даты курс нельзя начать.">
              <UInput v-model="startsAt" type="datetime-local" size="lg" class="w-full" />
            </UFormField>
            <UFormField label="Пройти до" :description="deadlineHint" :error="deadlineError || undefined">
              <UInput v-model="deadlineDate" type="date" size="lg" class="w-full" :min="todayIso()" />
            </UFormField>
          </div>
        </section>

        <div class="flex flex-wrap gap-2">
          <UButton
            color="primary"
            size="lg"
            icon="i-lucide-user-plus"
            :loading="previewing && !confirmOpen"
            :disabled="!canSubmit"
            @click="askAssign"
          >
            Назначить…
          </UButton>
          <UButton
            color="neutral"
            variant="ghost"
            size="lg"
            :to="{ name: 'admin-course-workspace', params: { courseId } }"
          >
            К курсу
          </UButton>
        </div>
        <p v-if="!hasRecipients" class="text-sm text-muted -mt-2">
          Выберите сотрудников, подразделение или «Все сотрудники».
        </p>
      </div>
    </div>

    <UModal
      v-model:open="confirmOpen"
      :title="`Назначить «${title}»?`"
      :description="previewCount
        ? `Получат курс: ${previewCount} ${plural(previewCount, ['сотрудник', 'сотрудника', 'сотрудников'])}. Кому курс уже назначен, повторно не назначается.`
        : 'По выбранным условиям получателей не нашлось.'"
    >
      <template #body>
        <div class="flex flex-col gap-3">
          <p v-if="allStaff" class="text-sm text-muted">
            Назначение постоянное: новые сотрудники получат курс при первом входе.
            <template v-if="previewWithoutOfo">
              Из получателей {{ previewWithoutOfo }} ещё не выбрали подразделение — скорее всего, не входили на портал.
            </template>
          </p>
          <p v-if="deadlineDate" class="text-sm text-muted">
            Срок: до {{ formatDate(toIsoOrNull(deadlineDate, true)) }} включительно.
          </p>
          <ul
            v-if="previewList.length"
            class="max-h-64 overflow-y-auto flex flex-col list-none p-0 m-0 rounded-lg ring-1 ring-inset ring-default divide-y divide-default"
          >
            <li v-for="(r, i) in previewList" :key="r.id ?? r.userId ?? i" class="px-3 py-2 text-sm text-default">
              {{ recipientName(r) }}
            </li>
          </ul>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton color="neutral" variant="ghost" @click="confirmOpen = false">Отмена</UButton>
          <UButton
            color="primary"
            icon="i-lucide-user-plus"
            :loading="assigning"
            :disabled="!previewCount"
            @click="onAssign"
          >
            Назначить
          </UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
