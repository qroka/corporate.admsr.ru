<script setup lang="ts">
import { ref, nextTick, computed, onMounted } from 'vue'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
  error?: boolean
}

const WELCOME_MESSAGE = 'Привет! Я корпоративный AI-ассистент ADMSR. Задавайте любые рабочие вопросы — помогу разобраться!'

const messages = ref<Message[]>([
  {
    id: 'welcome',
    role: 'assistant',
    content: WELCOME_MESSAGE,
    timestamp: new Date(),
  },
])

const inputText = ref('')
const isLoading = ref(false)
const messagesEl = ref<HTMLElement | null>(null)

const canSend = computed(() => inputText.value.trim().length > 0 && !isLoading.value)

const suggestions = [
  'Как подать заявку на отпуск?',
  'Какие корпоративные мероприятия запланированы?',
  'Как оформить больничный?',
  'Расскажи о корпоративных ценностях',
]

function formatTime(date: Date) {
  return date.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

async function scrollToBottom(smooth = true) {
  await nextTick()
  if (messagesEl.value) {
    messagesEl.value.scrollTo({
      top: messagesEl.value.scrollHeight,
      behavior: smooth ? 'smooth' : 'instant',
    })
  }
}

function clearChat() {
  messages.value = [
    {
      id: 'welcome',
      role: 'assistant',
      content: WELCOME_MESSAGE,
      timestamp: new Date(),
    },
  ]
}

async function sendMessage(text?: string) {
  const content = (text ?? inputText.value).trim()
  if (!content || isLoading.value) return

  inputText.value = ''

  messages.value.push({
    id: `u-${Date.now()}`,
    role: 'user',
    content,
    timestamp: new Date(),
  })
  await scrollToBottom()

  isLoading.value = true

  try {
    const history = messages.value
      .filter(m => !m.error && m.id !== 'welcome')
      .slice(-20)
      .map(m => ({ role: m.role, content: m.content }))

    const res = await fetch('/api/chat.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ messages: history }),
    })

    const data = await res.json()

    messages.value.push({
      id: `a-${Date.now()}`,
      role: 'assistant',
      content: data.success ? data.message : (data.error || 'Произошла ошибка. Попробуйте снова.'),
      timestamp: new Date(),
      error: !data.success,
    })
  } catch {
    messages.value.push({
      id: `e-${Date.now()}`,
      role: 'assistant',
      content: 'Не удалось подключиться к серверу. Проверьте соединение.',
      timestamp: new Date(),
      error: true,
    })
  } finally {
    isLoading.value = false
    await scrollToBottom()
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

onMounted(() => scrollToBottom(false))
</script>

<template>
  <div class="flex flex-col h-[calc(100vh-64px)] bg-default">

    <!-- ─── Header ──────────────────────────────────────────── -->
    <div class="flex items-center justify-between px-6 py-3 border-b border-default bg-default shrink-0">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-primary flex items-center justify-center shadow-sm">
          <UIcon name="i-lucide-bot" class="text-white size-5" />
        </div>
        <div>
          <p class="font-semibold text-highlighted leading-tight">AI Ассистент</p>
          <div class="flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse" />
            <span class="text-xs text-muted">Онлайн · llama-3.1-8b</span>
          </div>
        </div>
      </div>

      <UTooltip text="Очистить историю чата">
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-trash-2"
          size="sm"
          @click="clearChat"
        />
      </UTooltip>
    </div>

    <!-- ─── Messages ───────────────────────────────────────── -->
    <div
      ref="messagesEl"
      class="flex-1 overflow-y-auto px-4 py-6 space-y-5 scroll-smooth"
    >
      <template v-for="msg in messages" :key="msg.id">

        <!-- Bot message -->
        <div v-if="msg.role === 'assistant'" class="flex items-end gap-2.5 max-w-3xl">
          <div class="w-8 h-8 rounded-xl bg-primary shrink-0 flex items-center justify-center mb-5">
            <UIcon name="i-lucide-bot" class="text-white size-4" />
          </div>
          <div class="flex flex-col gap-1">
            <div
              class="px-4 py-3 rounded-2xl rounded-bl-sm text-sm leading-relaxed wrap-break-word whitespace-pre-wrap"
              :class="msg.error
                ? 'bg-red-50 dark:bg-red-950/40 text-red-700 dark:text-red-300 border border-red-200 dark:border-red-800/60'
                : 'bg-elevated text-default border border-default'"
            >
              <UIcon
                v-if="msg.error"
                name="i-lucide-alert-circle"
                class="inline mr-1.5 text-red-500 size-4 align-text-bottom"
              />{{ msg.content }}
            </div>
            <span class="text-[11px] text-muted pl-1">{{ formatTime(msg.timestamp) }}</span>
          </div>
        </div>

        <!-- User message -->
        <div v-else class="flex items-end gap-2.5 flex-row-reverse ml-auto max-w-3xl">
          <div class="w-8 h-8 rounded-xl bg-elevated border border-default shrink-0 flex items-center justify-center mb-5">
            <UIcon name="i-lucide-user" class="text-muted size-4" />
          </div>
          <div class="flex flex-col gap-1 items-end">
            <div class="px-4 py-3 rounded-2xl rounded-br-sm text-sm leading-relaxed bg-primary text-white wrap-break-word whitespace-pre-wrap">
              {{ msg.content }}
            </div>
            <span class="text-[11px] text-muted pr-1">{{ formatTime(msg.timestamp) }}</span>
          </div>
        </div>

      </template>

      <!-- Typing indicator -->
      <div v-if="isLoading" class="flex items-end gap-2.5 max-w-3xl">
        <div class="w-8 h-8 rounded-xl bg-primary shrink-0 flex items-center justify-center">
          <UIcon name="i-lucide-bot" class="text-white size-4" />
        </div>
        <div class="px-4 py-3.5 rounded-2xl rounded-bl-sm bg-elevated border border-default flex items-center gap-1.5">
          <span class="w-2 h-2 rounded-full bg-muted animate-bounce [animation-delay:0ms]" />
          <span class="w-2 h-2 rounded-full bg-muted animate-bounce [animation-delay:150ms]" />
          <span class="w-2 h-2 rounded-full bg-muted animate-bounce [animation-delay:300ms]" />
        </div>
      </div>

      <!-- Suggestions (shown only on welcome state) -->
      <div v-if="messages.length === 1 && !isLoading" class="pt-2">
        <p class="text-xs text-muted mb-3 pl-11">Быстрые вопросы:</p>
        <div class="flex flex-wrap gap-2 pl-11">
          <UButton
            v-for="s in suggestions"
            :key="s"
            color="neutral"
            variant="outline"
            size="xs"
            class="rounded-full"
            @click="sendMessage(s)"
          >
            {{ s }}
          </UButton>
        </div>
      </div>
    </div>

    <!-- ─── Input ───────────────────────────────────────────── -->
    <div class="shrink-0 px-4 pb-4 pt-3 border-t border-default bg-default">
      <div class="flex items-end gap-3 max-w-4xl mx-auto">
        <UTextarea
          v-model="inputText"
          placeholder="Напишите сообщение..."
          :rows="1"
          autoresize
          :maxrows="5"
          class="flex-1"
          :disabled="isLoading"
          @keydown="handleKeydown"
        />
        <UButton
          icon="i-lucide-send-horizontal"
          :disabled="!canSend"
          :loading="isLoading"
          class="shrink-0"
          @click="sendMessage()"
        >
          Отправить
        </UButton>
      </div>
      <p class="text-[11px] text-muted text-center mt-2">
        <kbd class="px-1 py-0.5 rounded border border-default text-[10px]">Enter</kbd>
        — отправить &nbsp;·&nbsp;
        <kbd class="px-1 py-0.5 rounded border border-default text-[10px]">Shift+Enter</kbd>
        — новая строка
      </p>
    </div>

  </div>
</template>
