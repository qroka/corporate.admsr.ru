<script setup lang="ts">
import * as z from 'zod'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { FormSubmitEvent } from '@nuxt/ui'
import { userNeedsOnboarding } from '../composables/useOnboarding'

const router = useRouter()

const schema = z.object({
  login: z.string().min(1, 'Введите логин'),
  password: z.string().min(1, 'Введите пароль'),
})

type Schema = z.output<typeof schema>

const state = reactive({
  login: '',
  password: '',
})

const loading = ref(false)
const errorMessage = ref('')

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  errorMessage.value = ''

  try {
    const res = await fetch('/api/auth.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ login: event.data.login, password: event.data.password }),
    })

    const data = await res.json()

    if (data.success) {
      localStorage.setItem('auth-user', JSON.stringify(data.user))
      const needsWelcome = await userNeedsOnboarding(data.user.id)
      await router.push(needsWelcome ? { name: 'onboarding' } : { name: 'home' })
    } else {
      errorMessage.value = data.message || 'Ошибка авторизации'
    }
  } catch {
    errorMessage.value = 'Не удалось подключиться к серверу'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-(--ui-bg) px-4 py-11">
    <div class="flex w-full flex-col items-center gap-6">
      <!-- Заголовок -->
      <div class="flex flex-col items-center gap-1 text-center">
        <p
            class="text-[clamp(22px,3.5vw,36px)] leading-tight text-(--ui-text-highlighted)"
            style="font-family: 'Unbounded', sans-serif;"
        >
          <span class="font-bold">ADMSR</span>
          <span class="font-light"> | КОРПОРАТИВНЫЙ ПОРТАЛ</span>
        </p>
        <p
            class="text-[clamp(38px,6.5vw,64px)] font-black uppercase leading-none text-primary"
            style="font-family: 'Unbounded', sans-serif;"
        >
          Авторизация
        </p>
      </div>

      <!-- Форма -->
      <div class="w-full max-w-[384px]">
        <UCard>
          <UForm
              :schema="schema"
              :state="state"
              class="flex flex-col gap-4"
              @submit="onSubmit"
          >
            <UAlert
                v-if="errorMessage"
                color="error"
                variant="soft"
                :description="errorMessage"
                icon="i-lucide-circle-alert"
            />

            <UFormField
                label="Логин"
                name="login"
                required
            >
              <UInput
                  v-model="state.login"
                  type="text"
                  placeholder="Введите логин"
                  autocomplete="username"
                  class="w-full"
              />
            </UFormField>

            <UFormField
                label="Пароль"
                name="password"
                required
            >
              <UInput
                  v-model="state.password"
                  type="password"
                  placeholder="Введите пароль"
                  autocomplete="current-password"
                  class="w-full"
              />
            </UFormField>

            <UButton
                type="submit"
                block
                :loading="loading"
            >
              Войти
            </UButton>
          </UForm>
        </UCard>
      </div>
    </div>
  </div>
</template>