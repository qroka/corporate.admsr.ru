<script setup>
import { onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { currentRole, setRole } from '../stores/role';
import { attachAbsenceStorageSync, hasActiveAbsence } from '../stores/absenceJournal';
import { clearAuthStorage } from '../composables/useAuthSession';
import { useHeaderUser } from '../composables/useHeaderUser';

defineProps({
  isDark: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['toggle-theme']);
const route = useRoute();
const router = useRouter();
const { canToggleAdminRole } = useHeaderUser();

function navigate(name) {
  router.push({ name });
}

/** В режиме администратора — конструктор курсов, иначе «Мои курсы». */
function openCourses() {
  if (currentRole.value === 'admin') {
    navigate('admin-courses');
    return;
  }
  navigate('courses');
}

/** Включить админ-режим UI и открыть управление курсами. */
async function openAdminCourses() {
  if (canToggleAdminRole.value) setRole('admin');
  await Promise.resolve();
  navigate('admin-courses');
}

const isNewsActive = () =>
  route.name === 'news' || route.name === 'news-details';

const isCoursesActive = () =>
  typeof route.name === 'string'
  && (route.name === 'courses' || route.name.startsWith('course-') || route.name.startsWith('admin-course'));

const isAdminCoursesActive = () =>
  typeof route.name === 'string' && route.name.startsWith('admin-course');

async function logout() {
  try {
    const user = JSON.parse(localStorage.getItem('auth-user') || 'null');
    if (user?.id) {
      await fetch('/api/logout.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: user.id }),
      });
    }
  } finally {
    clearAuthStorage();
    router.push({ name: 'login' });
  }
}

onMounted(() => {
  attachAbsenceStorageSync();
});
</script>

<template>
  <UContainer as="aside" class="flex flex-col justify-between w-fit gap-0 z-0 mx-0">
    <!-- Верхняя группа: функциональные сервисы для сотрудников -->
    <UContainer class="flex flex-col w-fit gap-6 z-0 mx-0">
      <!-- Блок быстрых ссылок на основные сервисы -->
      <UContainer class="flex flex-col bg-elevated relative rounded-full w-fit gap-0 sm:p-3 md:p-3 lg:p-3 xl:p-3 z-0 mx-0">
        <UTooltip arrow :content="{side: 'right'}" text="Журнал отсутствия">
          <div class="relative">
            <UButton
              type="button"
              color="neutral"
              square
              class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
              :class="route.name === 'absence-journal'
                ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
                : ''"
              size="xl"
              icon="i-lucide-calendar"
              @click="navigate('absence-journal')"
            />

            <UChip
              v-if="hasActiveAbsence"
              color="primary"
              inset
              size="3xl"
              class="absolute -top-0.5 -right-0.5 z-30"
            />
          </div>
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="Новости">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="isNewsActive()
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-newspaper"
            @click="navigate('news')"
          />
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="Дни рождения коллег">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'birthdays'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-cake"
            @click="navigate('birthdays')"
          />
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="Заявки">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'applications'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-book-open"
            @click="navigate('applications')"
          />
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="База знаний">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'knowledge-base'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-book-open"
            @click="navigate('knowledge-base')"
          />
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="Кадровый резерв">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'personnel-reserve'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-users"
            @click="navigate('personnel-reserve')"
          />
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="Тесты">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'tests'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-clipboard-check"
            @click="navigate('tests')"
          />
        </UTooltip>

        <UTooltip
          arrow
          :content="{side: 'right'}"
          :text="currentRole === 'admin' ? 'Управление курсами' : 'Мои курсы'"
        >
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="isCoursesActive()
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-graduation-cap"
            aria-label="Курсы"
            @click="openCourses"
          />
        </UTooltip>
      </UContainer>

      <!-- Блок ссылок на подразделения HR -->
      <UContainer class="flex flex-col bg-elevated relative rounded-full w-fit gap-0 sm:p-3 md:p-3 lg:p-3 xl:p-3 z-0 mx-0">
        <UTooltip arrow :content="{side: 'right'}" text="Отдел кадров">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'hr-department'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-building-2"
            @click="navigate('hr-department')"
          />
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="Отдел муниципальной службы">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'municipal-service'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-building"
            @click="navigate('municipal-service')"
          />
        </UTooltip>

        <UTooltip arrow :content="{side: 'right'}" text="Отдел развития и мотивации персонала">
          <UButton
            type="button"
            color="neutral"
            square
            class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
            :class="route.name === 'development-motivation'
              ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
              : ''"
            size="xl"
            icon="i-lucide-sparkles"
            @click="navigate('development-motivation')"
          />
        </UTooltip>
      </UContainer>
    </UContainer>

    <!-- Нижний блок: служебные действия (тема, помощь, выход) -->
    <UContainer class="flex flex-col bg-elevated relative rounded-full w-fit gap-0 sm:p-3 md:p-3 lg:p-3 xl:p-3 z-0 mx-0">
      <!-- Переключатель светлой / тёмной темы -->
      <UTooltip arrow :content="{side: 'right'}" :text="isDark ? 'Светлая тема' : 'Тёмная тема'">
        <UButton
          type="button"
          color="neutral"
          square
          size="xl"
          class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
          :icon="isDark ? 'i-lucide-sun' : 'i-lucide-moon'"
          @click="emit('toggle-theme', $event)"
        />
      </UTooltip>

      <!-- Справка -->
      <UTooltip v-if="currentRole === 'admin'" arrow :content="{side: 'right'}" text="Дэшборд администратора">
        <UButton
          type="button"
          color="neutral"
          square
          size="xl"
          class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
          :class="route.name === 'admin'
            ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
            : ''"
          icon="i-lucide-layout-dashboard"
          @click="navigate('admin')"
        />
      </UTooltip>

      <UTooltip
        v-if="canToggleAdminRole || currentRole === 'admin'"
        arrow
        :content="{side: 'right'}"
        text="Управление курсами"
      >
        <UButton
          type="button"
          color="neutral"
          square
          size="xl"
          class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50  [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
          :class="isAdminCoursesActive()
            ? 'z-10 bg-primary text-neutral-50 shadow-none dark:shadow-brand [&_svg]:text-neutral-50 hover:bg-primary hover:text-neutral-50 active:bg-primary active:text-neutral-50 active:[&_svg]:text-neutral-50'
            : ''"
          icon="i-lucide-library-big"
          aria-label="Управление курсами"
          @click="openAdminCourses"
        />
      </UTooltip>

      <!-- Выход -->
      <UTooltip arrow :content="{side: 'right'}" text="Выход из личного кабинета">
        <UButton
          type="button"
          color="neutral"
          square
          size="xl"
          class="relative z-0 shadow-none transition-all cursor-pointer duration-300 ease-out rounded-full bg-accented text-toned hover:bg-neutral-900 hover:text-neutral-50 [&_svg]:text-dimmed hover:[&_svg]:text-neutral-50 active:[&_svg]:text-inverted active:text-inverted active:bg-inverted"
          icon="i-lucide-log-out"
          @click="logout"
        />
      </UTooltip>
    </UContainer>
  </UContainer>
</template>

