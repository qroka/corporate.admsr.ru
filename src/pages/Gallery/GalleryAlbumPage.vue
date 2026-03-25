<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';

type Photo = {
  id: string;
  src: string;
  title?: string;
  description?: string;
};

const route = useRoute();
const router = useRouter();

const albumId = computed(() => String(route.params.albumId ?? ''));

const albumTitle = computed(() => `Альбом ${albumId.value}`);
const albumDescription = computed(() => 'Фотографии с мероприятия. Нажмите на фото, чтобы открыть в большом размере.');

const modules = import.meta.glob('../../img/Events/*.{jpg,jpeg,png,webp}', { eager: true, import: 'default' }) as Record<
  string,
  string
>;

function numericKey(path: string) {
  const m = path.match(/(\d+)\.(jpg|jpeg|png|webp)$/i);
  return m ? Number(m[1]) : Number.POSITIVE_INFINITY;
}

const items = computed<Photo[]>(() => {
  return Object.entries(modules)
    .sort(([a], [b]) => numericKey(a) - numericKey(b))
    .map(([path, src], index) => {
      const filename = path.split('/').pop() ?? `photo-${index + 1}`;
      const n = index + 1;
      const withMeta = n % 5 === 0; // пример: не у всех фото есть метаданные
      return {
        id: filename,
        src,
        ...(withMeta
          ? {
              title: `Фото ${n}`,
              description: `Кадр ${n} из альбома ${albumId.value}.`,
            }
          : {}),
      };
    });
});

const viewportWidth = ref<number>(typeof window === 'undefined' ? 1280 : window.innerWidth);
function onResize() {
  viewportWidth.value = window.innerWidth;
}

onMounted(() => {
  window.addEventListener('resize', onResize, { passive: true });
});
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize);
});

const lanes = computed(() => (viewportWidth.value < 640 ? 1 : viewportWidth.value < 1280 ? 2 : 3));
const estimateSize = computed(() => (viewportWidth.value < 640 ? 420 : 480));

const selected = ref<Photo | null>(null);
const modalOpen = computed({
  get: () => selected.value !== null,
  set: (open: boolean) => {
    if (!open) selected.value = null;
  },
});

function openPhoto(p: Photo) {
  selected.value = p;
}
</script>

<template>
  <UMain class="flex flex-col w-full h-full min-h-0">
    <UContainer class="flex flex-col gap-4 sm:p-0 md:p-0 lg:p-0 xl:p-0 mx-0 shrink-0">
      <UPageHeader title="" class="border-none p-0 w-full">
        <template #title>
          <div class="flex items-center gap-3 min-w-0">
            <UButton
              type="button"
              color="neutral"
              variant="soft"
              icon="i-lucide-arrow-left"
              @click="router.push({ name: 'gallery' })"
            />
            <div class="min-w-0">
              <h1 class="text-2xl font-medium truncate">
                {{ albumTitle }}
              </h1>
              <p class="text-sm text-muted truncate">
                {{ albumDescription }}
              </p>
            </div>
          </div>
        </template>
      </UPageHeader>
    </UContainer>

    <UScrollArea
      v-slot="{ item, index }"
      :items="items"
      orientation="vertical"
      :virtualize="{
        gap: 12,
        lanes,
        estimateSize,
        overscan: 6
      }"
      class="flex-1 min-h-0 w-full"
    >
      <div class="rounded-xl overflow-hidden bg-elevated ring ring-transparent hover:ring-accented transition w-full">
        <button
          type="button"
          class="block w-full"
          @click="openPhoto(item)"
        >
          <img
            :src="item.src"
            :alt="item.title || 'Фотография'"
            :loading="index > 8 ? 'lazy' : 'eager'"
            decoding="async"
            class="block w-full h-auto"
          >
        </button>

        <!-- Метаданные как на макете: просто блок под фото -->
        <div v-if="item.title || item.description" class="border-t border-muted px-5 py-5">
          <h3 v-if="item.title" class="text-xl font-semibold text-highlighted leading-tight">
            {{ item.title }}
          </h3>
          <p v-if="item.description" class="mt-3 text-base text-muted whitespace-pre-line">
            {{ item.description }}
          </p>
        </div>
      </div>
    </UScrollArea>

    <UModal
      v-model:open="modalOpen"
      class="w-[96vw] max-w-7xl h-[88vh] p-0"
      :ui="{ content: 'p-0', header: 'p-0', body: 'p-0', footer: 'p-0' }"
    >
      <template #content="{ close }">
        <div
          v-if="selected"
          class="flex h-full w-full min-h-0"
          :class="(selected.title || selected.description) ? '' : 'bg-elevated/50'"
        >
          <!-- body (слева) -->
          <div class="flex-1 min-w-0 min-h-0 p-4 flex items-center justify-center bg-elevated/50">
            <img
              :src="selected.src"
              :alt="selected.title || 'Фотография'"
              class="max-h-full max-w-full w-auto h-auto object-contain rounded-lg"
              decoding="async"
            />
          </div>

          <!-- если нет метаданных — показываем только картинку -->
          <div v-if="selected.title || selected.description" class="w-96 shrink-0 min-h-0 border-l border-muted bg-default p-6 flex flex-col gap-4">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-lg font-semibold text-highlighted truncate">
                  {{ selected.title || 'Без названия' }}
                </h2>
              </div>
              <UButton
                type="button"
                color="neutral"
                variant="ghost"
                icon="i-lucide-x"
                square
                @click="close()"
              />
            </div>
            <p v-if="selected.description" class="text-sm text-muted whitespace-pre-line">
              {{ selected.description }}
            </p>
          </div>

          <!-- кнопка закрытия, когда показываем только картинку -->
          <UButton
            v-else
            type="button"
            color="neutral"
            variant="ghost"
            icon="i-lucide-x"
            square
            class="absolute top-3 right-3"
            @click="close()"
          />
        </div>
      </template>
    </UModal>
  </UMain>
</template>

