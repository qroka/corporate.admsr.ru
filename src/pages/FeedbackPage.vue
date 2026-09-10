<script setup lang="ts">
import { reactive, ref } from 'vue';
import { apiSessionFetch } from '../composables/useAuthSession';
import { useAppToast } from '../composables/useAppToast';

const { success, error } = useAppToast();

const categoryOptions = [
  { label: 'Вопрос', value: 'question' },
  { label: 'Ошибка / баг', value: 'bug' },
  { label: 'Идея / предложение', value: 'idea' },
  { label: 'Другое', value: 'other' },
];

const form = reactive({
  category: 'question',
  subject: '',
  message: '',
});

const errors = reactive({
  subject: '',
  message: '',
});

const submitting = ref(false);
const sent = ref(false);

function validate() {
  errors.subject = '';
  errors.message = '';
  let ok = true;
  if (!form.subject.trim()) {
    errors.subject = 'Укажите тему';
    ok = false;
  } else if (form.subject.trim().length > 200) {
    errors.subject = 'Тема слишком длинная (до 200 символов)';
    ok = false;
  }
  if (!form.message.trim()) {
    errors.message = 'Напишите сообщение';
    ok = false;
  } else if (form.message.trim().length < 10) {
    errors.message = 'Сообщение слишком короткое (минимум 10 символов)';
    ok = false;
  } else if (form.message.trim().length > 5000) {
    errors.message = 'Сообщение слишком длинное (до 5000 символов)';
    ok = false;
  }
  return ok;
}

async function onSubmit() {
  if (!validate() || submitting.value) return;
  submitting.value = true;
  try {
    const res = await apiSessionFetch('/api/feedback.php', {
      method: 'POST',
      json: {
        category: form.category,
        subject: form.subject.trim(),
        message: form.message.trim(),
      },
    });
    if (!res.success) {
      error('Не удалось отправить', res.message || 'Попробуйте позже');
      return;
    }
    sent.value = true;
    form.subject = '';
    form.message = '';
    form.category = 'question';
    success('Сообщение отправлено', 'Спасибо! Мы ответим при необходимости.');
  } catch (e: any) {
    error('Не удалось отправить', e?.message || 'Ошибка сети');
  } finally {
    submitting.value = false;
  }
}

function writeAgain() {
  sent.value = false;
}
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0 overflow-y-auto scrollbar-hide">
    <div class="flex flex-col gap-4 w-full max-w-xl mx-auto pb-10">
      <UPageHeader
        title="Обратная связь"
        description="Сообщите об ошибке, задайте вопрос или предложите улучшение"
      />

      <UCard v-if="sent" variant="outline" class="bg-elevated">
        <div class="flex flex-col items-start gap-4">
          <div class="flex items-center gap-3">
            <div class="size-10 rounded-full bg-success/10 grid place-items-center">
              <UIcon name="i-lucide-check" class="size-5 text-success" />
            </div>
            <div>
              <p class="font-semibold text-highlighted">Сообщение принято</p>
              <p class="text-sm text-muted">Команда портала получит ваше обращение.</p>
            </div>
          </div>
          <UButton color="neutral" variant="outline" @click="writeAgain">
            Написать ещё
          </UButton>
        </div>
      </UCard>

      <UCard v-else variant="outline" class="bg-elevated">
        <UForm class="flex flex-col gap-4" @submit.prevent="onSubmit">
          <UFormField label="Тип обращения" name="category">
            <USelect
              v-model="form.category"
              :items="categoryOptions"
              class="w-full"
            />
          </UFormField>

          <UFormField label="Тема" name="subject" :error="errors.subject" required>
            <UInput
              v-model="form.subject"
              placeholder="Кратко, о чём речь"
              maxlength="200"
              class="w-full"
            />
          </UFormField>

          <UFormField label="Сообщение" name="message" :error="errors.message" required>
            <UTextarea
              v-model="form.message"
              placeholder="Опишите ситуацию подробнее…"
              :rows="6"
              autoresize
              maxlength="5000"
              class="w-full"
            />
          </UFormField>

          <div class="flex items-center justify-end gap-2 pt-1">
            <UButton
              type="submit"
              color="primary"
              :loading="submitting"
              :disabled="submitting"
              icon="i-lucide-send"
            >
              Отправить
            </UButton>
          </div>
        </UForm>
      </UCard>
    </div>
  </UMain>
</template>
