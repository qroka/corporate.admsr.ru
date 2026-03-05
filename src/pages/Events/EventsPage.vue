<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch, reactive } from 'vue';
import type { BlogPostProps, EditorToolbarItem } from '@nuxt/ui';
import { currentRole } from '../../stores/role';
import rawEvents from '../../data/events.json';

type EventPost = BlogPostProps & {
  badge?: string;
};

const posts = ref<EventPost[]>(
  (rawEvents as EventPost[]).map((event, index) => ({
    ...event,
    to: `/events/${index}`,
    target: undefined,
  })),
);

const searchQuery = ref('');
const selectedBadges = ref<string[]>([]);
const dateFrom = ref('');
const dateTo = ref('');
const filtersOpen = ref(false);
const inputDate = ref();
// Используем any для значения UInputDate range, чтобы не конфликтовать с внутренним типом CalendarDate
type DateRangeValue = { start?: any; end?: any } | null;
const modelValue = ref<DateRangeValue>(null);

// Slideover для создания мероприятия (локальный режим, без сохранения)
const createOpen = ref(false);

type CreateFormState = {
  title: string;
  description: string;
  badge: string | null;
  date: string;
  image: string;
  link: string;
};

const createState = reactive<CreateFormState>({
  title: '',
  description: '',
  badge: null,
  date: '',
  image: '/src/img/tailwindcss-v4.svg',
  link: '#',
});

type SingleDateValue = { value?: any } | any | null;
const createDateValue = ref<SingleDateValue>(null);
const createSubmitting = ref(false);
const createError = ref<string | null>(null);

// Панель инструментов редактора новости (описание мероприятия)
const editorToolbarItems: EditorToolbarItem[][] = [
  [
    {
      icon: 'i-lucide-heading',
      tooltip: { text: 'Заголовки' },
      content: { align: 'start' },
      items: [
        { kind: 'heading', level: 1, icon: 'i-lucide-heading-1', label: 'Заголовок 1' },
        { kind: 'heading', level: 2, icon: 'i-lucide-heading-2', label: 'Заголовок 2' },
        { kind: 'heading', level: 3, icon: 'i-lucide-heading-3', label: 'Заголовок 3' },
        { kind: 'heading', level: 4, icon: 'i-lucide-heading-4', label: 'Заголовок 4' },
      ],
    },
  ],
  [
    { kind: 'mark', mark: 'bold', icon: 'i-lucide-bold', tooltip: { text: 'Жирный' } },
    { kind: 'mark', mark: 'italic', icon: 'i-lucide-italic', tooltip: { text: 'Курсив' } },
    { kind: 'mark', mark: 'underline', icon: 'i-lucide-underline', tooltip: { text: 'Подчёркнутый' } },
    { kind: 'mark', mark: 'strike', icon: 'i-lucide-strikethrough', tooltip: { text: 'Зачёркнутый' } },
    { kind: 'mark', mark: 'code', icon: 'i-lucide-code', tooltip: { text: 'Код' } },
  ],
  [
    { kind: 'bulletList', icon: 'i-lucide-list', tooltip: { text: 'Маркированный список' } },
    { kind: 'orderedList', icon: 'i-lucide-list-ordered', tooltip: { text: 'Нумерованный список' } },
  ],
  [
    {
      icon: 'i-lucide-align-justify',
      tooltip: { text: 'Выравнивание' },
      content: { align: 'end' },
      items: [
        { kind: 'textAlign', align: 'left', icon: 'i-lucide-align-left', label: 'По левому краю' },
        { kind: 'textAlign', align: 'center', icon: 'i-lucide-align-center', label: 'По центру' },
        { kind: 'textAlign', align: 'right', icon: 'i-lucide-align-right', label: 'По правому краю' },
        { kind: 'textAlign', align: 'justify', icon: 'i-lucide-align-justify', label: 'По ширине' },
      ],
    },
  ],
  [
    { kind: 'link', icon: 'i-lucide-link', tooltip: { text: 'Ссылка' } },
    { kind: 'image', icon: 'i-lucide-image', tooltip: { text: 'Изображение по URL' } },
  ],
];

const uniqueBadges = computed(() => {
  const set = new Set<string>();
  posts.value.forEach((p) => p.badge && set.add(p.badge));
  return Array.from(set).sort();
});
const badgeOptions = computed(() =>
  uniqueBadges.value.map((b) => ({ value: b, label: b })),
);

const filteredPosts = computed(() => {
  let list = posts.value;

  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase();
    list = list.filter((p) => {
      const title = p.title ?? '';
      const description = p.description ?? '';
      return title.toLowerCase().includes(q) || description.toLowerCase().includes(q);
    });
  }

  if (selectedBadges.value.length > 0) {
    list = list.filter((p) => p.badge && selectedBadges.value.includes(p.badge));
  }

  if (dateFrom.value) {
    list = list.filter((p) => p.date && p.date >= dateFrom.value);
  }
  if (dateTo.value) {
    list = list.filter((p) => p.date && p.date <= dateTo.value);
  }

  return list;
});

function resetFilters() {
  searchQuery.value = '';
  selectedBadges.value = [];
  dateFrom.value = '';
  dateTo.value = '';
  modelValue.value = null;
  page.value = 1;
}

function resetCreateForm() {
  createState.title = '';
  createState.description = '';
  createState.badge = null;
  createState.date = '';
  createState.image = '/src/img/tailwindcss-v4.svg';
  createState.link = '#';
  createDateValue.value = null;
  createError.value = null;
}

function createSyncDate(val: SingleDateValue) {
  const d = val?.value ?? val;
  if (d && typeof d.toString === 'function') {
    createState.date = d.toString();
  } else {
    createState.date = '';
  }
}

function validateCreate(): boolean {
  if (!createState.title.trim()) {
    createError.value = 'Заполните название мероприятия.';
    return false;
  }
  if (!createState.date) {
    createError.value = 'Выберите дату проведения.';
    return false;
  }
  createError.value = null;
  return true;
}

async function handleCreateSubmit() {
  if (!validateCreate()) return;
  createSubmitting.value = true;
  try {
    // Локальный режим: добавляем мероприятие только в текущий список (без сохранения в файл).
    posts.value.push({
      title: createState.title,
      description: createState.description,
      badge: createState.badge ?? undefined,
      date: createState.date,
      image: createState.image,
      to: undefined,
    } as EventPost);
    createOpen.value = false;
  } finally {
    createSubmitting.value = false;
  }
}

const activeFilterCount = computed(() => {
  let count = selectedBadges.value.length;
  if (dateFrom.value) count += 1;
  if (dateTo.value) count += 1;
  return count;
});

const page = ref(1);

const sm = ref(false);
const md = ref(false);
const lg = ref(false);

const itemsPerPage = computed(() => {
  if (lg.value) return 8;
  if (md.value) return 6;
  if (sm.value) return 4;
  return 4; // grid-cols-1 — 4 элемента на странице
});

watch(
  modelValue,
  (val) => {
    const toStringSafe = (d: any) =>
      d && typeof d.toString === 'function' ? d.toString() : '';

    dateFrom.value = val?.start ? toStringSafe(val.start) : '';
    dateTo.value = val?.end ? toStringSafe(val.end) : '';
  },
  { immediate: true }
);

onMounted(() => {
  const qSm = window.matchMedia('(min-width: 640px)');
  const qMd = window.matchMedia('(min-width: 768px)');
  const qLg = window.matchMedia('(min-width: 1024px)');
  const update = () => {
    sm.value = qSm.matches;
    md.value = qMd.matches;
    lg.value = qLg.matches;
  };
  update();
  qSm.addEventListener('change', update);
  qMd.addEventListener('change', update);
  qLg.addEventListener('change', update);
  onUnmounted(() => {
    qSm.removeEventListener('change', update);
    qMd.removeEventListener('change', update);
    qLg.removeEventListener('change', update);
  });
});

watch([itemsPerPage, filteredPosts], ([newSize, _], [oldSize]) => {
  if (oldSize === undefined) return;
  const total = filteredPosts.value.length;
  const maxPage = Math.max(1, Math.ceil(total / itemsPerPage.value));
  if (page.value > maxPage) page.value = maxPage;
});

watch(
  () => [searchQuery.value, selectedBadges.value, dateFrom.value, dateTo.value],
  () => {
    page.value = 1;
  }
);

const paginatedPosts = computed(() => {
  const start = (page.value - 1) * itemsPerPage.value;
  return filteredPosts.value.slice(start, start + itemsPerPage.value);
});
</script>

<template>
  <content class="flex flex-col gap-6 w-full h-full min-h-0">
    <Headline class="text-zinc-700 dark:text-zinc-50">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between w-full">
        <div>
          <h1 class="text-4xl leading-12 font-display">Мероприятия</h1>
          <p class="text-xl leading-6 text-zinc-500">
            Здесь будет лента ближайших корпоративных мероприятий.
          </p>
        </div>
        <UButton
          v-if="currentRole === 'admin'"
          color="primary"
          size="xl"
          class="w-full sm:w-auto justify-center"
          @click="
            resetCreateForm();
            createOpen = true;
          "
        >
          Добавить мероприятие
        </UButton>
      </div>
    </Headline>
    <UMain class="flex flex-1 min-h-0 flex-col w-full h-full gap-6">
      <!-- Обычный поиск + кнопка фильтров -->
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
        <UInput v-model="searchQuery" icon="i-lucide-search" size="xl" color="neutral" variant="outline"
          placeholder="Поиск по названию..." class="w-full" />
        <UButton @click="filtersOpen = true" color="neutral" variant="outline" size="xl" class="justify-center">
          Фильтры
          <span v-if="activeFilterCount > 0"
            class="inline-flex items-center justify-center rounded-full bg-zinc-900 text-zinc-50 dark:bg-zinc-50 dark:text-zinc-700 text-xs h-6 w-6">
            {{ activeFilterCount }}
          </span>
        </UButton>
      </div>
      <p v-if="filteredPosts.length !== posts.length" class="text-sm text-zinc-500 mb-2">
        Найдено: {{ filteredPosts.length }} из {{ posts.length }}
      </p>
      <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3 flex-1 min-h-0">
        <UBlogPost
          v-for="post in paginatedPosts"
          :key="post.title"
          v-bind="post"
          class="h-full"
        />
      </div>
      <div class="flex justify-center">
        <UPagination size="xl" v-model:page="page" :total="filteredPosts.length" :items-per-page="itemsPerPage" />
      </div>
      <!-- Slideover с фильтрами по бейджам и датам -->
      <USlideover v-model:open="filtersOpen" side="right" title="Фильтры">
        <template #body>
          <div class="flex flex-col gap-3 py-4">
            <h3 class="text-sm font-medium text-zinc-700 dark:text-zinc-200">
              Категория
            </h3>
            <USelect placeholder="Выберите категорию" size="xl" v-model="selectedBadges" multiple :items="badgeOptions"
              class="w-full" />
            <h3 class="text-sm font-medium text-zinc-700 dark:text-zinc-200">
              Даты проведения
            </h3>
            <UInputDate class="w-full" size="xl" ref="inputDate" v-model="modelValue" range>
              <template #trailing>
                <UPopover :reference="inputDate?.inputsRef[0]?.$el">
                  <UButton color="neutral" variant="link" size="sm" icon="i-lucide-calendar"
                    aria-label="Select a date range" class="px-0" />
                  <template #content>
                    <UCalendar v-model="modelValue" class="p-2" :number-of-months="2" range />
                  </template>
                </UPopover>
              </template>
            </UInputDate>
          </div>
        </template>
        <template #footer>
          <div class="flex justify-between gap-3 items-center w-full">
            <UButton @click="resetFilters" color="neutral" variant="outline" size="xl" class="w-full justify-center">
              Сбросить
            </UButton>
            <UButton @click="filtersOpen = false" size="xl" class="w-full justify-center">
              Применить
            </UButton>
          </div>
        </template>
      </USlideover>
      <!-- Slideover создания мероприятия (только для администратора) -->
      <USlideover v-model:open="createOpen" side="right" title="Новое мероприятие">
        <template #body>
          <div class="flex flex-col gap-4 py-2">
            <UForm :state="createState" class="space-y-4" @submit.prevent="handleCreateSubmit">
              <UFormGroup label="Название" name="title" required>
                <UInput v-model="createState.title" size="lg" placeholder="Например, Семинар по цифровым сервисам" />
              </UFormGroup>

              <UFormGroup label="Категория (бейдж)" name="badge">
                <USelect
                  v-model="createState.badge"
                  :items="badgeOptions"
                  placeholder="Выберите категорию"
                  size="lg"
                  class="w-full"
                />
              </UFormGroup>

              <UFormGroup label="Описание" name="description">
                <div class="rounded-md border border-zinc-200 dark:border-zinc-700 overflow-auto h-full">
                  <UEditor
                    v-slot="{ editor }"
                    v-model="createState.description"
                    content-type="markdown"
                    placeholder="Опишите цель и формат мероприятия. Поддерживаются заголовки, списки, ссылки и изображения по URL."
                    :ui="{ base: 'p-3 min-h-40' }"
                    class="w-full h-full "
                  >
                    <UEditorToolbar
                      :editor="editor"
                      :items="editorToolbarItems"
                      class="border-b border-zinc-200 dark:border-zinc-700 py-2 px-3 overflow-x-auto bg-zinc-50 dark:bg-zinc-800/50"
                    />
                  </UEditor>
                </div>
              </UFormGroup>

              <UFormGroup label="Дата проведения" name="date" required>
                <UInputDate
                  v-model="createDateValue"
                  size="lg"
                  class="w-full"
                  @update:model-value="createSyncDate"
                >
                  <template #trailing>
                    <UPopover>
                      <UButton
                        color="neutral"
                        variant="link"
                        size="sm"
                        icon="i-lucide-calendar"
                        aria-label="Выбрать дату"
                        class="px-0"
                      />
                      <template #content>
                        <UCalendar v-model="createDateValue" class="p-2" />
                      </template>
                    </UPopover>
                  </template>
                </UInputDate>
              </UFormGroup>

              <UFormGroup label="Ссылка на подробности" name="link">
                <UInput
                  v-model="createState.link"
                  size="lg"
                  placeholder="https://..."
                />
              </UFormGroup>

              <UFormGroup label="Изображение (URL)" name="image">
                <UInput v-model="createState.image" size="lg" placeholder="/src/img/event-cover.svg" />
              </UFormGroup>

              <UAlert
                v-if="createError"
                color="red"
                variant="subtle"
                icon="i-lucide-alert-circle"
              >
                {{ createError }}
              </UAlert>
            </UForm>
          </div>
        </template>
        <template #footer>
          <div class="flex justify-between gap-3 items-center w-full">
            <UButton
              color="neutral"
              variant="outline"
              size="xl"
              class="w-full justify-center"
              @click="createOpen = false"
            >
              Отмена
            </UButton>
            <UButton
              size="xl"
              class="w-full justify-center"
              :loading="createSubmitting"
              @click="handleCreateSubmit"
            >
              Создать
            </UButton>
          </div>
        </template>
      </USlideover>
    </UMain>
  </content>
</template>

