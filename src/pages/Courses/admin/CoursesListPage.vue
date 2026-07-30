<script setup lang="ts">
/**
 * Совместимость: /admin/courses → /courses?tab=manage
 * Управление видно только при включённом переключателе администратора.
 */
import { onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useSectionAccess } from '../../../composables/useSectionAccess';

const router = useRouter();
const { canEditSection, ensureLoaded } = useSectionAccess();

onMounted(async () => {
  await ensureLoaded();
  if (canEditSection('courses')) {
    router.replace({ name: 'courses', query: { tab: 'manage' } });
  } else {
    router.replace({ name: 'courses' });
  }
});
</script>

<template>
  <UMain class="flex flex-1 items-center justify-center">
    <USkeleton class="h-10 w-48 rounded-lg" />
  </UMain>
</template>
