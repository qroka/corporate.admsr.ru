<template>
  <div class="app-root h-screen overflow-hidden">
    <UApp :locale="ru">
      <!-- Основной лейаут приложения: вертикальный флекс-контейнер на высоту экрана -->
      <div class="h-screen w-full max-w-[1600px] mx-auto overflow-hidden flex flex-col justify-start p-6 gap-6 transition-colors"
        :class="containerClass">
        <AppHeader
          :is-dark="isDark"
          :active-nav="activeNav"
          @toggle-theme="startThemeTransition"

        />

        <!-- Основной контент: слева вертикальное меню (только xl+), справа область под страницы -->
        <main class="flex flex-1 w-full h-full gap-6 min-h-0">
          <!-- Левая колонка с быстрыми действиями (скрыта ниже xl, дублируется в бургер-меню) -->
          <aside class="hidden xl:flex flex-col justify-between shrink-0 z-20">
            <!-- Верхняя группа: функциональные сервисы для сотрудников -->
            <aside class="flex flex-col">
              <!-- Блок быстрых ссылок на основные сервисы -->
              <aside class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-18 flex flex-col items-center p-3">
                <RouterLink :to="{ name: 'absence-journal' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'absence-journal'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <CalendarDaysIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'absence-journal'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Журнал отсутствия
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'applications' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'applications'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BookOpenIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'applications'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Заявки
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'knowledge-base' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'knowledge-base'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BookOpenIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'knowledge-base'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    База знаний
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'personnel-reserve' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'personnel-reserve'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <UserGroupIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'personnel-reserve'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Кадровый резерв
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'tests' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'tests'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <ClipboardDocumentCheckIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'tests'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Тесты
                  </span>
                </RouterLink>
              </aside>


              <!-- Блок ссылок на подразделения HR -->
              <aside class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-18 flex flex-col items-center p-3">
                <RouterLink :to="{ name: 'hr-department' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'hr-department'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BuildingOffice2Icon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'hr-department'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Отдел кадров
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'municipal-service' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'municipal-service'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <BuildingOfficeIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'municipal-service'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Отдел муниципальной службы
                  </span>
                </RouterLink>
                <RouterLink :to="{ name: 'development-motivation' }"
                  class="group self-start flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out"
                  :class="route.name === 'development-motivation'
                    ? 'bg-brand text-zinc-50 shadow-none z-10 dark:shadow-brand'
                    : 'bg-zinc-200 dark:bg-zinc-700 text-zinc-400 hover:bg-zinc-900 '">
                  <SparklesIcon class="w-6 h-6 shrink-0" />
                  <span
                    class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                    :class="route.name === 'development-motivation'
                      ? 'text-zinc-50'
                      : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                    Отдел развития и мотивации персонала
                  </span>
                </RouterLink>
              </aside>
            </aside>

            <!-- Нижний блок: служебные действия (тема, помощь, выход) -->
            <aside class="bg-zinc-100 dark:bg-zinc-800 rounded-full w-18 flex flex-col items-center p-3">
              <!-- Переключатель светлой / тёмной темы -->
              <button type="button"
                class="btn-icon group self-start z-10 bg-zinc-200 dark:bg-zinc-700 text-zinc-400 flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out hover:bg-zinc-900"
                @click="startThemeTransition">
                <span class="relative w-6 h-6 shrink-0">
                  <!-- Луна -->
                  <MoonIcon class="absolute inset-0 w-6 h-6 transition-all duration-300 ease-out"
                    :class="isDark ? 'opacity-0 -rotate-90 scale-75' : 'opacity-100 rotate-0 scale-100'" />
                  <!-- Солнце -->
                  <SunIcon class="absolute inset-0 w-6 h-6 transition-all duration-300 ease-out"
                    :class="isDark ? 'opacity-100 rotate-0 scale-100' : 'opacity-0 rotate-90 scale-75'" />
                </span>
                <span
                  class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                  :class="isDark ? 'text-zinc-50' : 'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                  {{ isDark ? 'Светлая тема' : 'Тёмная тема' }}
                </span>
              </button>
              <!-- Справка -->
              <button type="button"
                class="btn-icon group self-start z-10 bg-zinc-200 dark:bg-zinc-700 text-zinc-400 flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out hover:bg-zinc-900">
                <QuestionMarkCircleIcon class="w-6 h-6 shrink-0" />
                <span
                  class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                  :class="'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                  Справка
                </span>
              </button>
              <!-- Выход -->
              <button type="button"
                class="btn-icon group self-start z-10 bg-zinc-200 dark:bg-zinc-700 text-zinc-400 flex items-center h-12 px-3 rounded-full cursor-pointer overflow-hidden min-w-12 w-max max-w-12 hover:max-w-[24rem] transition-[max-width,background-color,color,transform] duration-300 ease-out hover:bg-zinc-900">
                <ArrowRightOnRectangleIcon class="w-6 h-6 shrink-0" />
                <span
                  class="whitespace-nowrap ml-0 opacity-0 transition-[margin,color,opacity] duration-300 ease-out group-hover:ml-2 group-hover:opacity-100"
                  :class="'text-zinc-700 group-hover:text-zinc-50 dark:text-zinc-50'">
                  Выход из личного кабинета
                </span>
              </button>
            </aside>
          </aside>

          <!-- Область для контента страниц, подключённых через Vue Router -->
          <section class="flex-1 min-w-0 min-h-0 overflow-y-auto max-h-full" >
            <RouterView />
          </section>
        </main>
      </div>
    </UApp>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { ru } from '@nuxt/ui/locale';
import { useRoute } from 'vue-router';
import AppHeader from './components/AppHeader.vue';
import {
  ArrowRightOnRectangleIcon,
  BookOpenIcon,
  BuildingOffice2Icon,
  BuildingOfficeIcon,
  CalendarDaysIcon,
  ClipboardDocumentCheckIcon,
  MoonIcon,
  QuestionMarkCircleIcon,
  SparklesIcon,
  SunIcon,
  UserGroupIcon,
} from '@heroicons/vue/24/outline';

// Текущий маршрут (используется для подсветки активных пунктов навигации)
const route = useRoute();

// Имя активного пункта верхнего меню (events / gallery / newcomers / culture)
const activeNav = computed(() => (route.name ?? 'events'));


// --- Поддержка светлой / тёмной темы для Nuxt UI ---
const COLOR_MODE_KEY = 'ui-color-mode';
const isDark = ref(false);

const applyColorMode = (mode) => {
  const root = document.documentElement;
  if (!root) return;

  if (mode === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
};

onMounted(() => {
  const saved = localStorage.getItem(COLOR_MODE_KEY);
  let mode;

  if (saved === 'light' || saved === 'dark') {
    mode = saved;
  } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    mode = 'dark';
  } else {
    mode = 'light';
  }

  isDark.value = mode === 'dark';
  applyColorMode(mode);
});

watch(isDark, (value) => {
  const mode = value ? 'dark' : 'light';
  localStorage.setItem(COLOR_MODE_KEY, mode);
  applyColorMode(mode);
});

const toggleColorMode = () => {
  isDark.value = !isDark.value;
};

// Анимированная смена темы с помощью View Transitions API
const startThemeTransition = (event) => {
  const anyDoc = document;

  if (!anyDoc.startViewTransition) {
    toggleColorMode();
    return;
  }

  const x = event.clientX;
  const y = event.clientY;
  const endRadius =
    Math.hypot(
      Math.max(x, window.innerWidth - x),
      Math.max(y, window.innerHeight - y),
    ) + 8;

  const transition = anyDoc.startViewTransition(() => {
    toggleColorMode();
  });

  transition.ready.then(() => {
    const duration = 600;
    anyDoc.documentElement.animate(
      {
        clipPath: [
          `circle(0px at ${x}px ${y}px)`,
          `circle(${endRadius}px at ${x}px ${y}px)`,
        ],
      },
      {
        duration,
        easing: 'cubic-bezier(.76,.32,.29,.99)',
        pseudoElement: '::view-transition-new(root)',
      },
    );
  });
};

// Класс фона/текста для корневого контейнера (визуальное переключение темы)
const containerClass = computed(() => 'bg-(--ui-bg) text-(--ui-text-highlighted)');
</script>

<style>
::view-transition-old(root),
::view-transition-new(root) {
  animation: none;
  mix-blend-mode: normal;
}

::view-transition-new(root) {
  z-index: 9999;
}

::view-transition-old(root) {
  z-index: 1;
}
</style>
