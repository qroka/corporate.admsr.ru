<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useSectionAccess } from '../composables/useSectionAccess';
import { currentRole } from '../stores/role';
import { useAppToast } from '../composables/useAppToast';
import {
  usePortalServices,
  type PortalService,
  type PortalServiceKind,
} from '../composables/usePortalServices';
import {
  PORTAL_SERVICE_CATALOG,
  PORTAL_SERVICE_ICON_OPTIONS,
  catalogItemByKey,
} from './Services/portalServiceCatalog';

const { toast } = useAppToast();
const { isSuperAdmin, ensureLoaded: ensureAccess } = useSectionAccess();
ensureAccess();

const canEditServices = computed(
  () => isSuperAdmin.value && currentRole.value === 'admin',
);

const {
  services,
  enabledServices,
  loading,
  error,
  ensureLoaded,
  createService,
  updateService,
  deleteService,
  reorderServices,
} = usePortalServices();

const displayServices = computed(() =>
  canEditServices.value ? services.value : enabledServices.value,
);

const slideOpen = ref(false);
const saving = ref(false);
const deleteOpen = ref(false);
const deleteTarget = ref<PortalService | null>(null);
const deleting = ref(false);
const reordering = ref(false);

const form = reactive({
  id: null as number | null,
  kind: 'internal' as PortalServiceKind,
  internalKey: '' as string,
  label: '',
  description: '',
  icon: 'i-lucide-layout-grid',
  path: '',
  externalUrl: '',
  isEnabled: true,
});

const catalogItems = computed(() =>
  PORTAL_SERVICE_CATALOG.map((c) => ({
    label: c.label,
    value: c.key,
    description: c.description,
    icon: c.icon,
  })),
);

const kindItems = [
  { label: 'Страница портала', value: 'internal' },
  { label: 'Внешняя ссылка', value: 'external' },
];

const iconItems = PORTAL_SERVICE_ICON_OPTIONS.map((icon) => ({
  label: icon.replace('i-lucide-', ''),
  value: icon,
  icon,
}));

onMounted(async () => {
  await ensureAccess();
  await ensureLoaded({ all: canEditServices.value });
});

function resetForm() {
  form.id = null;
  form.kind = 'internal';
  form.internalKey = '';
  form.label = '';
  form.description = '';
  form.icon = 'i-lucide-layout-grid';
  form.path = '';
  form.externalUrl = '';
  form.isEnabled = true;
}

function openCreate() {
  resetForm();
  slideOpen.value = true;
}

function openEdit(s: PortalService) {
  form.id = s.id;
  form.kind = s.kind;
  form.internalKey = s.internalKey || '';
  form.label = s.label;
  form.description = s.description || '';
  form.icon = s.icon || 'i-lucide-layout-grid';
  form.path = s.path || '';
  form.externalUrl = s.externalUrl || '';
  form.isEnabled = s.isEnabled !== false;
  slideOpen.value = true;
}

function onCatalogPick(key: string | null) {
  if (!key) return;
  const item = catalogItemByKey(key);
  if (!item) return;
  form.internalKey = item.key;
  form.label = item.label;
  form.description = item.description;
  form.icon = item.icon;
  form.path = item.path;
}

async function onSave() {
  if (form.kind === 'internal') {
    if (!form.internalKey && !form.path) {
      toast.add({
        title: 'Выберите страницу портала',
        color: 'warning',
        icon: 'i-lucide-alert-triangle',
      });
      return;
    }
    if (!form.label.trim()) {
      toast.add({ title: 'Укажите название', color: 'warning', icon: 'i-lucide-alert-triangle' });
      return;
    }
  } else {
    if (!form.label.trim() || !form.externalUrl.trim()) {
      toast.add({
        title: 'Укажите название и ссылку',
        color: 'warning',
        icon: 'i-lucide-alert-triangle',
      });
      return;
    }
  }

  saving.value = true;
  try {
    const payload = {
      kind: form.kind,
      label: form.label.trim(),
      description: form.description.trim(),
      icon: form.icon,
      isEnabled: form.isEnabled,
      internalKey: form.kind === 'internal' ? form.internalKey || null : null,
      path: form.kind === 'internal' ? form.path || null : null,
      externalUrl: form.kind === 'external' ? form.externalUrl.trim() : null,
    };
    if (form.id != null && form.id > 0) {
      await updateService({ ...payload, id: form.id, sortOrder: services.value.find((s) => s.id === form.id)?.sortOrder });
      toast.add({ title: 'Сервис сохранён', color: 'success', icon: 'i-lucide-check' });
    } else {
      await createService(payload);
      toast.add({ title: 'Сервис добавлен', color: 'success', icon: 'i-lucide-check' });
    }
    slideOpen.value = false;
  } catch (e: any) {
    toast.add({
      title: 'Не удалось сохранить',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-x',
    });
  } finally {
    saving.value = false;
  }
}

function askDelete(s: PortalService) {
  deleteTarget.value = s;
  deleteOpen.value = true;
}

async function confirmDelete() {
  if (!deleteTarget.value || deleteTarget.value.id <= 0) return;
  deleting.value = true;
  try {
    await deleteService(deleteTarget.value.id);
    toast.add({ title: 'Сервис удалён', color: 'success', icon: 'i-lucide-check' });
    deleteOpen.value = false;
    deleteTarget.value = null;
  } catch (e: any) {
    toast.add({
      title: 'Не удалось удалить',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-x',
    });
  } finally {
    deleting.value = false;
  }
}

async function moveService(index: number, dir: -1 | 1) {
  const list = [...displayServices.value];
  const j = index + dir;
  if (j < 0 || j >= list.length) return;
  if (list.some((s) => s.id <= 0)) {
    toast.add({
      title: 'Сначала дождитесь загрузки с сервера',
      color: 'warning',
      icon: 'i-lucide-alert-triangle',
    });
    return;
  }
  const tmp = list[index];
  list[index] = list[j];
  list[j] = tmp;
  reordering.value = true;
  try {
    await reorderServices(list.map((s) => s.id));
  } catch (e: any) {
    toast.add({
      title: 'Не удалось изменить порядок',
      description: e?.message,
      color: 'error',
      icon: 'i-lucide-x',
    });
  } finally {
    reordering.value = false;
  }
}

function cardProps(s: PortalService) {
  if (s.kind === 'external' && s.externalUrl) {
    return {
      to: s.externalUrl,
      target: '_blank' as const,
      rel: 'noopener noreferrer',
      external: true,
    };
  }
  return { to: s.path || '/services' };
}
</script>

<template>
  <div class="flex flex-col gap-6 w-full max-w-[1600px] mx-auto">
    <UPageHeader
      headline="Сервисы"
      title="Все сервисы"
      description="Рабочие системы и информационные системы в одном месте"
    >
      <template v-if="canEditServices" #links>
        <UButton color="primary" icon="i-lucide-plus" @click="openCreate">
          Добавить сервис
        </UButton>
      </template>
    </UPageHeader>

    <div v-if="loading && !displayServices.length" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <USkeleton v-for="n in 3" :key="n" class="h-28 w-full rounded-panel" />
    </div>

    <UAlert
      v-else-if="error && !displayServices.length"
      color="warning"
      variant="subtle"
      icon="i-lucide-alert-circle"
      title="Не удалось загрузить сервисы"
      :description="error"
    />

    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
      <div
        v-for="(service, index) in displayServices"
        :key="service.id"
        class="relative group"
      >
        <UPageCard
          :title="service.label"
          :description="service.description || undefined"
          :icon="service.icon"
          variant="soft"
          class="bg-elevated h-full"
          v-bind="cardProps(service)"
        />
        <div
          v-if="canEditServices && service.id > 0"
          class="absolute top-2 right-2 flex gap-1 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
        >
          <UButton
            color="neutral"
            variant="soft"
            size="xs"
            icon="i-lucide-arrow-up"
            square
            :disabled="reordering || index === 0"
            aria-label="Выше"
            @click.prevent="moveService(index, -1)"
          />
          <UButton
            color="neutral"
            variant="soft"
            size="xs"
            icon="i-lucide-arrow-down"
            square
            :disabled="reordering || index === displayServices.length - 1"
            aria-label="Ниже"
            @click.prevent="moveService(index, 1)"
          />
          <UButton
            color="neutral"
            variant="soft"
            size="xs"
            icon="i-lucide-pencil"
            square
            aria-label="Редактировать"
            @click.prevent="openEdit(service)"
          />
          <UButton
            color="error"
            variant="soft"
            size="xs"
            icon="i-lucide-trash-2"
            square
            aria-label="Удалить"
            @click.prevent="askDelete(service)"
          />
        </div>
        <UBadge
          v-if="canEditServices && !service.isEnabled"
          color="neutral"
          variant="subtle"
          size="sm"
          class="absolute bottom-3 left-3"
        >
          Скрыт
        </UBadge>
        <UBadge
          v-if="canEditServices && service.kind === 'external'"
          color="primary"
          variant="subtle"
          size="sm"
          class="absolute bottom-3 right-3"
        >
          Внешний
        </UBadge>
      </div>
    </div>

    <USlideover
      v-model:open="slideOpen"
      :title="form.id ? 'Редактировать сервис' : 'Новый сервис'"
      description="Внутренняя страница портала или внешняя ссылка"
    >
      <template #body>
        <div class="flex flex-col gap-4">
          <UFormField label="Тип">
            <USelectMenu
              v-model="form.kind"
              :items="kindItems"
              value-key="value"
              label-key="label"
              class="w-full"
              :search-input="false"
            />
          </UFormField>

          <UFormField v-if="form.kind === 'internal'" label="Страница портала" required>
            <USelectMenu
              v-model="form.internalKey"
              :items="catalogItems"
              value-key="value"
              label-key="label"
              placeholder="Выберите из сайдбара"
              class="w-full"
              @update:model-value="onCatalogPick"
            />
          </UFormField>

          <UFormField label="Название" required>
            <UInput v-model="form.label" class="w-full" size="lg" />
          </UFormField>

          <UFormField label="Описание">
            <UTextarea v-model="form.description" :rows="2" class="w-full" />
          </UFormField>

          <UFormField label="Иконка">
            <USelectMenu
              v-model="form.icon"
              :items="iconItems"
              value-key="value"
              label-key="label"
              class="w-full"
            >
              <template #leading="{ modelValue }">
                <UIcon v-if="modelValue" :name="String(modelValue)" class="size-4" />
              </template>
              <template #item-leading="{ item }">
                <UIcon :name="item.value" class="size-4" />
              </template>
            </USelectMenu>
          </UFormField>

          <UFormField v-if="form.kind === 'external'" label="Ссылка" required>
            <UInput
              v-model="form.externalUrl"
              type="url"
              placeholder="https://…"
              class="w-full"
              size="lg"
            />
          </UFormField>

          <UFormField v-if="canEditServices" label="Видимость">
            <USwitch v-model="form.isEnabled" label="Показывать пользователям" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex gap-2 justify-end w-full">
          <UButton color="neutral" variant="ghost" @click="slideOpen = false">Отмена</UButton>
          <UButton color="primary" :loading="saving" icon="i-lucide-check" @click="onSave">
            Сохранить
          </UButton>
        </div>
      </template>
    </USlideover>

    <UModal
      v-model:open="deleteOpen"
      title="Удалить сервис?"
      description="Карточка исчезнет со страницы сервисов, с главной и из сайдбара."
    >
      <template #body>
        <p class="text-sm text-muted">
          {{ deleteTarget?.label }}
        </p>
      </template>
      <template #footer>
        <div class="flex gap-2 justify-end w-full">
          <UButton color="neutral" variant="ghost" @click="deleteOpen = false">Отмена</UButton>
          <UButton color="error" :loading="deleting" icon="i-lucide-trash-2" @click="confirmDelete">
            Удалить
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>
