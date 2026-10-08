<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import {
  BIRTHDAY_GREETING_MAX_LENGTH,
  BIRTHDAY_GREETING_TEMPLATES,
  greetingAddressName,
  useBirthdayGreetings,
} from '../composables/useBirthdayGreetings';
import { useAppToast } from '../composables/useAppToast';

/**
 * Поздравление с днём рождения: свой текст или готовая фраза, публикуется на
 * стене именинника. Открывается через useBirthdayGreetings().openGreeting —
 * один экземпляр на странице.
 */
const { target, slideoverOpen, sendGreeting } = useBirthdayGreetings();
const { success, error } = useAppToast();

const text = ref('');
const sending = ref(false);

const addressName = computed(() => greetingAddressName(target.value?.name ?? ''));
const templates = computed(() =>
  BIRTHDAY_GREETING_TEMPLATES.map((t) => t.replaceAll('{name}', addressName.value)),
);
const canSend = computed(() => {
  const s = text.value.trim();
  return !!s && s.length <= BIRTHDAY_GREETING_MAX_LENGTH && !sending.value;
});

function pickTemplate(phrase: string) {
  text.value = phrase;
}

function close() {
  slideoverOpen.value = false;
}

watch(slideoverOpen, (open) => {
  if (open) text.value = '';
});

async function submit() {
  if (!canSend.value || !target.value) return;
  sending.value = true;
  const name = target.value.name;
  try {
    await sendGreeting(text.value);
    slideoverOpen.value = false;
    success('Поздравление опубликовано', `Оно появилось на стене: ${name}.`);
  } catch (e) {
    error('Не удалось поздравить', e instanceof Error ? e.message : undefined);
  } finally {
    sending.value = false;
  }
}
</script>

<template>
  <USlideover
    v-model:open="slideoverOpen"
    side="right"
    title="Поздравить с днём рождения"
    description="Поздравление появится на стене именинника, его увидят коллеги"
  >
    <template #body>
      <form id="birthday-greeting-form" class="flex flex-col gap-6" @submit.prevent="submit">
        <UUser
          v-if="target"
          :name="target.name"
          description="День рождения"
          :avatar="{ src: target.avatar, alt: target.name, icon: 'i-lucide-user' }"
          size="lg"
        />

        <div class="flex flex-col gap-2">
          <p class="text-sm font-medium text-highlighted">Готовые фразы</p>
          <p class="text-xs text-muted">Нажмите, чтобы подставить в поле, затем при желании поправьте.</p>
          <div class="flex flex-col gap-1.5">
            <UButton
              v-for="phrase in templates"
              :key="phrase"
              type="button"
              color="neutral"
              :variant="text === phrase ? 'soft' : 'outline'"
              size="sm"
              :label="phrase"
              :ui="{ base: 'justify-start text-left', label: 'whitespace-normal overflow-visible' }"
              @click="pickTemplate(phrase)"
            />
          </div>
        </div>

        <UFormField label="Текст поздравления" name="greeting" required :hint="`${text.length} / ${BIRTHDAY_GREETING_MAX_LENGTH}`">
          <UTextarea
            v-model="text"
            :rows="5"
            autoresize
            :maxrows="12"
            :maxlength="BIRTHDAY_GREETING_MAX_LENGTH"
            placeholder="Напишите несколько тёплых слов"
            class="w-full"
          />
        </UFormField>
      </form>
    </template>

    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="ghost" :disabled="sending" @click="close">Отмена</UButton>
        <UButton
          type="submit"
          form="birthday-greeting-form"
          color="primary"
          icon="i-lucide-gift"
          :loading="sending"
          :disabled="!canSend"
        >
          Поздравить
        </UButton>
      </div>
    </template>
  </USlideover>
</template>
