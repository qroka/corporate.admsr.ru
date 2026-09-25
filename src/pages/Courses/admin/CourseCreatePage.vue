<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { z } from 'zod';
import { useCoursesStore } from '../../../composables/useCoursesStore';
import { useAppToast } from '../../../composables/useAppToast';
import { useSectionAccess } from '../../../composables/useSectionAccess';

const router = useRouter();
const store = useCoursesStore();
const { toast } = useAppToast();
const { allowedCourseCategoryItems, loaded: accessLoaded, ensureLoaded } = useSectionAccess();
ensureLoaded();

const saving = ref(false);

const schema = z.object({
  title: z.string().trim().min(1, 'Укажите название обучения'),
  category: z.string().min(1, 'Выберите категорию'),
  shortDescription: z.string().trim().max(300, 'Не длиннее 300 символов'),
});

const form = reactive({
  title: '',
  category: '',
  shortDescription: '',
  sequentialProgress: true,
});

/** Категорий всего две, поэтому единственную доступную выбираем сами. */
watch(
  allowedCourseCategoryItems,
  (items) => {
    if (items.length === 1 && !form.category) form.category = items[0].value;
  },
  { immediate: true },
);

const noCategories = computed(() => accessLoaded.value && !allowedCourseCategoryItems.value.length);

async function onSubmit() {
  saving.value = true;
  try {
    const created = (await store.createCourse({
      title: form.title.trim(),
      category: form.category,
      shortDescription: form.shortDescription.trim(),
      sequentialProgress: form.sequentialProgress,
    })) as any;
    const id = created?.course?.id ?? created?.id;
    if (!id) throw new Error('Сервер не вернул id обучения');
    toast.add({
      title: 'Обучение создано',
      description: 'Шаг 2 из 4: добавьте первую тему',
      color: 'success',
      icon: 'i-lucide-check',
    });
    await router.push({
      name: 'admin-course-topic-create',
      params: { courseId: String(id) },
      query: { guide: '1' },
    });
  } catch (e: any) {
    toast.add({ title: 'Не удалось создать', description: e?.message, color: 'error', icon: 'i-lucide-x' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <UMain class="relative w-full h-full min-h-0">
    <div class="flex flex-col gap-6 w-full h-full min-h-0 max-w-3xl mx-auto overflow-y-auto scrollbar-hide p-px pb-8">
      <UPageHeader
        headline="Обучение"
        title="Новое обучение"
        description="Название и категория — остальное настроим на следующих шагах"
      />

      <UAlert
        color="primary"
        variant="subtle"
        icon="i-lucide-list-ordered"
        title="Шаг 1 из 4 — обучение"
        description="После сохранения откроется добавление первой темы, затем материал и тест."
      />

      <!-- Без доступных категорий форму отправить невозможно — говорим об этом прямо -->
      <UAlert
        v-if="noCategories"
        color="warning"
        variant="subtle"
        icon="i-lucide-lock"
        title="Нет доступных категорий"
        description="Создавать обучение можно только в своей категории. Обратитесь к администратору портала, чтобы вам её выдали."
      >
        <template #actions>
          <UButton color="neutral" variant="soft" :to="{ name: 'admin-courses' }">
            К списку обучения
          </UButton>
        </template>
      </UAlert>

      <UForm v-else :schema="schema" :state="form" class="flex flex-col gap-4" @submit="onSubmit">
        <UFormField label="Название" name="title" required>
          <UInput
            v-model="form.title"
            size="lg"
            class="w-full"
            autofocus
            placeholder="Например: Онбординг новых сотрудников"
          />
        </UFormField>

        <UFormField label="Категория" name="category" required>
          <USelectMenu
            v-model="form.category"
            :items="allowedCourseCategoryItems"
            value-key="value"
            label-key="label"
            placeholder="Выберите категорию"
            size="lg"
            color="neutral"
            :search-input="false"
            class="w-full"
            :content="{ align: 'start', sideOffset: 8 }"
          />
        </UFormField>

        <UFormField
          label="Краткое описание"
          name="shortDescription"
          hint="Необязательно"
          description="Одна-две строки — их видно в списке обучения."
        >
          <UTextarea
            v-model="form.shortDescription"
            :rows="3"
            :maxlength="300"
            class="w-full"
            placeholder="О чём этот курс и кому он нужен"
          />
        </UFormField>

        <UFormField
          label="Последовательное прохождение"
          name="sequentialProgress"
          description="Выключите, если темы можно изучать в любом порядке."
        >
          <USwitch v-model="form.sequentialProgress" label="Темы открываются по порядку" />
        </UFormField>

        <div class="flex items-center gap-2 pt-2">
          <UButton type="submit" color="primary" size="lg" :loading="saving" icon="i-lucide-arrow-right">
            Создать и добавить тему
          </UButton>
          <UButton color="neutral" variant="ghost" size="lg" :to="{ name: 'admin-courses' }">
            Отмена
          </UButton>
        </div>
      </UForm>
    </div>
  </UMain>
</template>
