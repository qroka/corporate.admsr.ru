<template>
  <section class="flex flex-col gap-6 text-zinc-700 dark:text-zinc-50">
    <header class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h1 class="text-4xl leading-12 font-display">Личный кабинет</h1>
        <p class="text-xl leading-6 text-zinc-500">
          Режим доступа влияет на доступные действия в интерфейсе.
        </p>
      </div>
      <div class="w-full max-w-md">
        <h2 class="text-sm font-medium text-zinc-600 dark:text-zinc-300 mb-2">
          Роль в интерфейсе
        </h2>
        <URadioGroup
          v-model="roleLocal"
          :items="roleOptions"
          orientation="horizontal"
          color="primary"
        />
      </div>
    </header>

    <div class="rounded-2xl border border-zinc-200 bg-white/70 p-4 dark:border-zinc-700 dark:bg-zinc-900/60">
      <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-100 mb-1">
        Текущий режим: {{ roleLabel }}
      </h3>
      <p class="text-sm text-zinc-500 dark:text-zinc-400">
        В режиме <span class="font-medium">Администратор</span> на странице «Мероприятия» доступна кнопка
        «Добавить мероприятие». В режиме <span class="font-medium">Пользователь</span> эта кнопка скрыта.
      </p>
    </div>
  </section>
</template>

<script setup>
import { computed, watch } from 'vue';
import { currentRole, setRole } from '../stores/role';

const roleOptions = [
  { value: 'user', label: 'Пользователь' },
  { value: 'admin', label: 'Администратор' },
];

const roleLocal = computed({
  get: () => currentRole.value,
  set: (val) => setRole(val),
});

const roleLabel = computed(() =>
  roleOptions.find((r) => r.value === currentRole.value)?.label || 'Пользователь',
);

// синхронизация на случай внешних изменений
watch(currentRole, (val) => {
  if (!roleOptions.some((r) => r.value === val)) {
    setRole('user');
  }
});
</script>

