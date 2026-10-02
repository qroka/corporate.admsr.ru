<script setup lang="ts">
/**
 * «Награды» — грамоты и достижения сотрудника. Видят все вошедшие; выдаёт и
 * удаляет администратор портала (кнопки не рендерятся без прав).
 * Сервер: profile_extras.php (award_add / award_delete).
 */
import { reactive, ref } from 'vue';
import { apiSessionFetch } from '../../composables/useAuthSession';
import { useAppToast } from '../../composables/useAppToast';
import ProfileSection from './ProfileSection.vue';

export type Award = { id: number; title: string; description: string; awardedOn: string; issuedBy: string };

const props = defineProps<{ userId: number; awards: Award[]; canManage: boolean }>();
const emit = defineEmits<{ (e: 'update', awards: Award[]): void }>();

const { success, error } = useAppToast();

const todayIso = () => new Date().toISOString().slice(0, 10);
const formOpen = ref(false);
const form = reactive({ title: '', description: '', awardedOn: todayIso() });
const saving = ref(false);
const removing = reactive<{ award: Award | null; busy: boolean }>({ award: null, busy: false });

function openForm() {
  Object.assign(form, { title: '', description: '', awardedOn: todayIso() });
  formOpen.value = true;
}

async function call(body: Record<string, unknown>, failTitle: string): Promise<Award[] | null> {
  try {
    const res = await apiSessionFetch<{ awards: Award[] }>('/api/profile_extras.php', { method: 'POST', json: body });
    if (!res?.success) {
      error(failTitle, res?.message);
      return null;
    }
    return res.data?.awards ?? [];
  } catch {
    error(failTitle, 'Проверьте подключение к сети и попробуйте ещё раз.');
    return null;
  }
}

async function issue() {
  if (!form.title.trim() || !form.awardedOn || saving.value) return;
  saving.value = true;
  const list = await call(
    { action: 'award_add', userId: props.userId, title: form.title.trim(), description: form.description.trim(), awardedOn: form.awardedOn },
    'Награда не выдана',
  );
  saving.value = false;
  if (list) {
    emit('update', list);
    formOpen.value = false;
    success('Награда выдана');
  }
}

async function confirmRemove() {
  const award = removing.award;
  if (!award) return;
  removing.busy = true;
  const list = await call({ action: 'award_delete', id: award.id }, 'Награда не удалена');
  removing.busy = false;
  if (list) {
    emit('update', list);
    removing.award = null;
  }
}

function formatDate(iso: string): string {
  const d = new Date(`${iso}T00:00:00`);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' });
}
</script>

<template>
  <ProfileSection v-if="awards.length || canManage" title="Награды" title-id="profile-awards">
    <template v-if="awards.length" #aside>{{ awards.length }}</template>

    <ul v-if="awards.length" class="flex flex-col divide-y divide-default">
      <li v-for="a in awards" :key="a.id" class="flex items-start gap-3 py-2.5 first:pt-0">
        <UIcon name="i-lucide-award" class="size-5 text-warning shrink-0 mt-0.5" aria-hidden="true" />
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-highlighted break-words">{{ a.title }}</p>
          <p v-if="a.description" class="text-sm text-muted whitespace-pre-line break-words">{{ a.description }}</p>
          <p class="text-xs text-dimmed">{{ formatDate(a.awardedOn) }}<template v-if="a.issuedBy"> · выдал(а) {{ a.issuedBy }}</template></p>
        </div>
        <UTooltip v-if="canManage" text="Удалить награду">
          <UButton type="button" color="neutral" variant="ghost" size="xs" square icon="i-lucide-trash-2" :aria-label="`Удалить награду: ${a.title}`" @click="removing.award = a" />
        </UTooltip>
      </li>
    </ul>
    <p v-else class="text-sm text-muted">Наград пока нет.</p>

    <UButton v-if="canManage" type="button" color="neutral" variant="outline" size="sm" icon="i-lucide-plus" class="mt-3" @click="openForm">
      Выдать награду
    </UButton>

    <UModal v-model:open="formOpen" title="Выдать награду" description="Грамота или достижение появится в профиле сотрудника">
      <template #body>
        <UForm id="award-form" :state="form" class="flex flex-col gap-4" @submit="issue">
          <UFormField label="Название" name="title" required>
            <UInput v-model="form.title" class="w-full" :maxlength="120" placeholder="Например, «Лучший специалист квартала»" />
          </UFormField>
          <UFormField label="Описание" name="description" hint="необязательно">
            <UTextarea v-model="form.description" class="w-full" :rows="3" autoresize :maxrows="8" :maxlength="500" />
          </UFormField>
          <UFormField label="Дата вручения" name="awardedOn" required>
            <UInput v-model="form.awardedOn" type="date" class="w-full" />
          </UFormField>
        </UForm>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" :disabled="saving" @click="formOpen = false">Отмена</UButton>
          <UButton type="submit" form="award-form" color="primary" :loading="saving" :disabled="!form.title.trim() || !form.awardedOn">Выдать</UButton>
        </div>
      </template>
    </UModal>

    <UModal
      :open="!!removing.award"
      title="Удалить награду?"
      :description="removing.award ? `«${removing.award.title}» исчезнет из профиля сотрудника.` : ''"
      @update:open="(v: boolean) => { if (!v && !removing.busy) removing.award = null; }"
    >
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" :disabled="removing.busy" @click="removing.award = null">Отмена</UButton>
          <UButton color="error" :loading="removing.busy" @click="confirmRemove">Удалить</UButton>
        </div>
      </template>
    </UModal>
  </ProfileSection>
</template>
