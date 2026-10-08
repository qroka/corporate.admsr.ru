<script setup lang="ts">
/**
 * Девблог (ADR-052): общий черновик администраторов в Markdown — слева текст,
 * справа предпросмотр так, как увидят сотрудники. Автосохранение с номером
 * версии: если черновик за это время сохранил другой администратор, правка
 * не перезаписывает его молча — показывается выбор. «Опубликовать» создаёт
 * новость «Девблог», окошко при входе и уведомление всем.
 *
 * Отдельная страница, а не USlideover: две колонки редактора и предпросмотра
 * в боковую панель не помещаются.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import type { TabsItem } from '@nuxt/ui';
import { useRouter } from 'vue-router';
import {
  DEVBLOG_BODY_MAX,
  DEVBLOG_DEFAULT_COVER,
  DEVBLOG_TITLE_MAX,
  DEVBLOG_VERSION_RE,
  loadDevblogDraft,
  publishDevblog,
  saveDevblogDraft,
  uploadDevblogCover,
  type DevblogDraft,
} from '../../composables/useDevblog';
import { useAppToast } from '../../composables/useAppToast';
import { useNewsData } from '../../composables/useNewsData';
import MarkdownEditor from '../../components/MarkdownEditor.vue';
import DevblogVersionBadge from '../../components/DevblogVersionBadge.vue';
import DevblogCover from '../../components/DevblogCover.vue';

const router = useRouter();
const { success, error } = useAppToast();
const { reload: reloadNews } = useNewsData();

const loading = ref(true);
const loadError = ref('');
const title = ref('');
const body = ref('');
const imagePath = ref('');
/** Версия выпуска; пустой черновик получает предложение сервера (последняя + 0.0.1). */
const releaseVersion = ref('');
const suggestedVersion = ref('1.0.0');
const version = ref(0);
const updatedAt = ref<string | null>(null);
const updatedBy = ref<DevblogDraft['updatedBy']>(null);

type SaveState = 'saved' | 'dirty' | 'saving' | 'error';
const saveState = ref<SaveState>('saved');
const saveError = ref('');
/** Черновик сохранил другой администратор — ждём решения, автосохранение стоит. */
const conflict = ref<DevblogDraft | null>(null);

const coverInput = ref<HTMLInputElement | null>(null);
const coverUploading = ref(false);

const previewRef = ref<{ editor?: { getHTML: () => string } } | null>(null);
const publishOpen = ref(false);
const publishing = ref(false);

const mobileTabs: TabsItem[] = [
  { label: 'Текст', value: 'edit', icon: 'i-lucide-pencil' },
  { label: 'Просмотр', value: 'preview', icon: 'i-lucide-eye' },
];
const mobileTab = ref<'edit' | 'preview'>('edit');

const coverSrc = computed(() => imagePath.value || DEVBLOG_DEFAULT_COVER);
const versionValid = computed(() => DEVBLOG_VERSION_RE.test(releaseVersion.value.trim()));
const canPublish = computed(
  () =>
    !!title.value.trim() &&
    !!body.value.trim() &&
    versionValid.value &&
    !conflict.value &&
    saveState.value !== 'saving',
);

const savedLabel = computed(() => {
  if (!updatedAt.value) return 'Черновик пуст';
  const t = new Date(updatedAt.value).toLocaleString('ru-RU', {
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  });
  return updatedBy.value ? `Сохранено · ${updatedBy.value.name}, ${t}` : `Сохранено · ${t}`;
});

/** Применить версию с сервера к полям; applying — чтобы watch не принял это за правку. */
let applying = false;
function applyDraft(d: DevblogDraft) {
  applying = true;
  title.value = d.title;
  body.value = d.body;
  imagePath.value = d.imagePath;
  suggestedVersion.value = d.suggestedVersion;
  releaseVersion.value = d.releaseVersion || d.suggestedVersion;
  version.value = d.version;
  updatedAt.value = d.updatedAt;
  updatedBy.value = d.updatedBy;
  saveState.value = 'saved';
  queueMicrotask(() => {
    applying = false;
  });
}

async function load() {
  loading.value = true;
  loadError.value = '';
  try {
    applyDraft(await loadDevblogDraft());
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : 'Не удалось загрузить черновик';
  } finally {
    loading.value = false;
  }
}

let saveTimer: ReturnType<typeof setTimeout> | null = null;
let saving: Promise<void> | null = null;

async function saveNow(force = false) {
  if (saveTimer) {
    clearTimeout(saveTimer);
    saveTimer = null;
  }
  if (conflict.value && !force) return;
  if (saving) await saving;
  saving = (async () => {
    saveState.value = 'saving';
    try {
      const res = await saveDevblogDraft(
        {
          title: title.value,
          body: body.value,
          imagePath: imagePath.value,
          // Недописанную версию («1.0.») не сохраняем — останется прежняя.
          releaseVersion: versionValid.value ? releaseVersion.value.trim() : '',
        },
        version.value,
        force,
      );
      if (res.ok) {
        version.value = res.draft.version;
        updatedAt.value = res.draft.updatedAt;
        updatedBy.value = res.draft.updatedBy;
        conflict.value = null;
        // Пока сохраняли, могли напечатать ещё — тогда остаёмся «есть изменения».
        saveState.value = saveTimer ? 'dirty' : 'saved';
      } else {
        conflict.value = res.draft;
        saveState.value = 'dirty';
      }
    } catch (e) {
      saveState.value = 'error';
      saveError.value = e instanceof Error ? e.message : 'Не удалось сохранить';
    } finally {
      saving = null;
    }
  })();
  return saving;
}

watch([title, body, imagePath, releaseVersion], () => {
  if (applying || loading.value) return;
  saveState.value = 'dirty';
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = setTimeout(() => void saveNow(), 1500);
});

/** Взять версию другого администратора — свои несохранённые правки пропадут. */
function takeTheirs() {
  if (!conflict.value) return;
  applyDraft(conflict.value);
  conflict.value = null;
}

/** Оставить свою — перезаписать черновик своей версией. */
function keepMine() {
  void saveNow(true);
}

/** Вернулись на вкладку и ничего не правили — подтянуть чужие изменения. */
async function onVisibility() {
  if (document.visibilityState !== 'visible' || saveState.value !== 'saved' || conflict.value) return;
  try {
    const d = await loadDevblogDraft();
    if (d.version !== version.value && saveState.value === 'saved') applyDraft(d);
  } catch {
    /* не страшно — проверим при сохранении */
  }
}

function pickCover() {
  coverInput.value?.click();
}

async function onCoverPicked(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (!file) return;
  coverUploading.value = true;
  try {
    imagePath.value = await uploadDevblogCover(file);
  } catch (err) {
    error('Не удалось загрузить обложку', err instanceof Error ? err.message : undefined);
  } finally {
    coverUploading.value = false;
  }
}

function resetCover() {
  imagePath.value = '';
}

function openPublish() {
  publishOpen.value = true;
}

function closePublish() {
  publishOpen.value = false;
}

async function confirmPublish(force = false) {
  const html = previewRef.value?.editor?.getHTML() ?? '';
  if (!html.trim()) {
    error('Не удалось подготовить текст', 'Откройте вкладку «Просмотр» и попробуйте ещё раз.');
    return;
  }
  publishing.value = true;
  try {
    await saveNow();
    const res = await publishDevblog(
      { title: title.value, body: body.value, imagePath: imagePath.value, releaseVersion: releaseVersion.value.trim(), html },
      version.value,
      force,
    );
    if ('conflict' in res) {
      conflict.value = res.draft;
      publishOpen.value = false;
      error('Черновик изменили', res.message);
      return;
    }
    publishOpen.value = false;
    applyDraft(res.draft);
    void reloadNews();
    success('Девблог опубликован', `Уведомление получили сотрудники: ${res.notified}.`);
    void router.push(`/news/${res.newsId}`);
  } catch (e) {
    error('Не удалось опубликовать', e instanceof Error ? e.message : undefined);
  } finally {
    publishing.value = false;
  }
}

onMounted(() => {
  void load();
  document.addEventListener('visibilitychange', onVisibility);
});

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibility);
  if (saveState.value === 'dirty' && !conflict.value) void saveNow();
});
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-[1600px] mx-auto overflow-y-auto scrollbar-hide p-px pb-8 *:shrink-0">
      <UPageHeader
        headline="Администрирование"
        title="Девблог"
        description="Общий черновик для всех администраторов. После публикации его увидят все сотрудники: окошко при входе, новость на рабочем столе и уведомление."
      >
        <template #links>
          <UButton
            color="primary"
            icon="i-lucide-send"
            label="Опубликовать"
            :disabled="!canPublish || loading"
            @click="openPublish"
          />
        </template>
      </UPageHeader>

      <div v-if="loading" class="flex flex-col gap-4">
        <USkeleton class="h-10 w-full max-w-xl" />
        <USkeleton class="h-40 w-64 rounded-panel" />
        <div class="grid gap-4 lg:grid-cols-2">
          <USkeleton class="h-[50vh] rounded-panel" />
          <USkeleton class="h-[50vh] rounded-panel" />
        </div>
      </div>

      <div v-else-if="loadError" class="flex flex-wrap items-center gap-3 text-sm">
        <span class="text-error">{{ loadError }}</span>
        <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-refresh-cw" @click="load">Повторить</UButton>
      </div>

      <template v-else>
        <UAlert
          v-if="conflict"
          color="warning"
          variant="subtle"
          icon="i-lucide-users"
          title="Черновик изменил другой администратор"
          :description="`Пока вы правили, свою версию сохранили: ${conflict.updatedBy?.name || 'другой администратор'}. Выберите, какую оставить.`"
          :actions="[
            { label: 'Взять сохранённую', color: 'neutral', variant: 'outline', onClick: takeTheirs },
            { label: 'Оставить мою', color: 'warning', onClick: keepMine },
          ]"
        />

        <div class="flex items-center gap-2 text-sm text-muted" aria-live="polite">
          <template v-if="saveState === 'saving'">
            <UIcon name="i-lucide-loader-circle" class="size-4 animate-spin" aria-hidden="true" />
            Сохраняем…
          </template>
          <template v-else-if="saveState === 'dirty'">
            <UIcon name="i-lucide-pencil" class="size-4" aria-hidden="true" />
            Есть несохранённые изменения
          </template>
          <template v-else-if="saveState === 'error'">
            <UIcon name="i-lucide-alert-circle" class="size-4 text-error" aria-hidden="true" />
            <span class="text-error">{{ saveError }}</span>
            <UButton size="xs" color="neutral" variant="link" class="p-0" @click="saveNow()">Повторить</UButton>
          </template>
          <template v-else>
            <UIcon name="i-lucide-cloud-check" class="size-4" aria-hidden="true" />
            {{ savedLabel }}
          </template>
        </div>

        <div class="grid gap-6 md:grid-cols-[minmax(0,1fr)_16rem] items-start">
          <div class="flex flex-col gap-4 min-w-0">
            <UFormField label="Заголовок" required :hint="`${title.length} / ${DEVBLOG_TITLE_MAX}`">
              <UInput
                v-model="title"
                size="lg"
                class="w-full"
                :maxlength="DEVBLOG_TITLE_MAX"
                placeholder="Например, «Обновление портала: комментарии и девблог»"
              />
            </UFormField>
            <UFormField
              label="Версия"
              required
              :hint="`Предложено: ${suggestedVersion}`"
              :error="releaseVersion && !versionValid ? 'В формате 1.0.0' : undefined"
              class="max-w-60"
            >
              <UInput
                v-model="releaseVersion"
                class="w-full tabular-nums"
                :placeholder="suggestedVersion"
                maxlength="14"
                inputmode="decimal"
              />
            </UFormField>
          </div>

          <UFormField label="Обложка" hint="Не выбрали — стандартная">
            <div class="flex flex-col gap-2">
              <img :src="coverSrc" alt="Обложка девблога" class="aspect-[16/10] w-full rounded-panel object-cover bg-elevated" />
              <div class="flex flex-wrap gap-2">
                <UButton
                  color="neutral"
                  variant="outline"
                  size="sm"
                  icon="i-lucide-upload"
                  :loading="coverUploading"
                  @click="pickCover"
                >
                  Загрузить
                </UButton>
                <UButton v-if="imagePath" color="neutral" variant="ghost" size="sm" icon="i-lucide-rotate-ccw" @click="resetCover">
                  Стандартная
                </UButton>
              </div>
              <input
                ref="coverInput"
                type="file"
                accept="image/jpeg,image/png,image/webp"
                class="sr-only"
                tabindex="-1"
                aria-hidden="true"
                @change="onCoverPicked"
              />
            </div>
          </UFormField>
        </div>

        <UTabs v-model="mobileTab" :items="mobileTabs" :content="false" variant="pill" size="sm" class="lg:hidden" />

        <div class="grid gap-6 lg:grid-cols-2 items-stretch">
          <div class="flex flex-col gap-2 min-w-0" :class="mobileTab === 'edit' ? '' : 'max-lg:hidden'">
            <div class="flex items-center justify-between gap-2">
              <p class="text-sm font-medium text-highlighted">Текст · Markdown</p>
              <UPopover :content="{ side: 'bottom', align: 'end' }">
                <UButton color="neutral" variant="link" size="xs" icon="i-lucide-circle-help" class="p-0">Шпаргалка</UButton>
                <template #content>
                  <div class="p-3 text-sm flex flex-col gap-1.5 font-mono text-toned">
                    <span># Заголовок, ## Подзаголовок</span>
                    <span>- пункт списка</span>
                    <span>1. нумерованный пункт</span>
                    <span>**жирный**, *курсив*, `код`</span>
                    <span>[текст ссылки](https://…)</span>
                    <span>&gt; цитата</span>
                    <span class="font-sans text-muted pt-1">Enter продолжает список, Tab — вложенность,</span>
                    <span class="font-sans text-muted">вставка адреса поверх текста — ссылка</span>
                  </div>
                </template>
              </UPopover>
            </div>
            <MarkdownEditor
              v-model="body"
              :rows="22"
              :maxlength="DEVBLOG_BODY_MAX"
              placeholder="## Что нового&#10;- Комментарии к новостям&#10;- Поздравления с днём рождения&#10;&#10;## Исправлено&#10;- Ввод времени в журнале отсутствия"
              aria-label="Текст девблога в Markdown"
            />
          </div>

          <div class="flex flex-col gap-2 min-w-0" :class="mobileTab === 'preview' ? '' : 'max-lg:hidden'">
            <p class="text-sm font-medium text-highlighted">Предпросмотр</p>
            <article class="flex-1 min-h-[50vh] rounded-panel border border-default bg-elevated/40 p-4 sm:p-6 flex flex-col gap-4">
              <UBadge color="primary" variant="subtle" class="self-start">Девблог</UBadge>
              <div class="flex flex-col gap-1">
                <h2 class="text-2xl font-bold text-highlighted text-pretty">{{ title || 'Без заголовка' }}</h2>
                <DevblogVersionBadge v-if="versionValid" :version="releaseVersion.trim()" />
              </div>
              <!-- Как увидят сотрудники в окошке: обложка между версией и текстом. -->
              <DevblogCover :src="coverSrc" />
              <UEditor
                ref="previewRef"
                :model-value="body"
                content-type="markdown"
                :editable="false"
                class="w-full"
                :ui="{ base: 'sm:px-0' }"
              />
              <UEmpty
                v-if="!body.trim()"
                variant="naked"
                icon="i-lucide-notebook-pen"
                title="Здесь будет девблог"
                description="Пишите слева — по пунктам, что нового и что исправлено."
              />
            </article>
          </div>
        </div>
      </template>
    </div>

    <UModal
      v-model:open="publishOpen"
      title="Опубликовать девблог?"
      description="Его увидят все сотрудники: окошко при следующем входе, новость «Девблог» на рабочем столе и уведомление. Черновик очистится."
    >
      <template #body>
        <p class="text-sm font-semibold text-highlighted">{{ title }}</p>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" :disabled="publishing" @click="closePublish">Отмена</UButton>
          <UButton color="primary" icon="i-lucide-send" :loading="publishing" @click="confirmPublish()">Опубликовать</UButton>
        </div>
      </template>
    </UModal>
  </UMain>
</template>
