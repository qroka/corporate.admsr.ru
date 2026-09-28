<script setup lang="ts">
/**
 * Место работы сотрудника: ОФО (ofo_unit) + должность из справочника этого ОФО.
 * Один компонент для профиля, онбординга и админки — чтобы данные сотрудника
 * везде заполнялись одинаково (раньше в админке ОФО вводили номером, а должность —
 * свободным текстом).
 *
 * Должность хранится строкой (user_info.role). Если текущей должности нет в
 * справочнике выбранного ОФО, она остаётся в списке с пометкой — открыть форму
 * и сохранить её без изменений не должно стирать должность.
 */
import { computed, ref, watch } from 'vue';
import { useOfoTree, type OfoPosition } from '../composables/useOfoTree';

const props = withDefaults(
  defineProps<{
    ofoId: number | null;
    role: string;
    required?: boolean;
    disabled?: boolean;
    size?: 'md' | 'lg' | 'xl';
    ofoHelp?: string;
  }>(),
  { required: false, disabled: false, size: 'lg', ofoHelp: undefined },
);

const emit = defineEmits<{
  (e: 'update:ofoId', v: number | null): void;
  (e: 'update:role', v: string): void;
}>();

const { ensureLoaded, error: ofoError, unitNumberOf, units, fetchPositions } = useOfoTree();
ensureLoaded();

const positions = ref<OfoPosition[]>([]);
const positionsLoading = ref(false);
let positionsSeq = 0;

async function loadPositions(ofoId: number | null) {
  const seq = ++positionsSeq;
  positions.value = [];
  const un = unitNumberOf(ofoId);
  if (un == null) return;
  positionsLoading.value = true;
  try {
    const list = await fetchPositions(un);
    if (seq === positionsSeq) positions.value = list;
  } catch {
    if (seq === positionsSeq) positions.value = [];
  } finally {
    if (seq === positionsSeq) positionsLoading.value = false;
  }
}

// Дерево ОФО грузится асинхронно: unit_number известен только после загрузки.
watch([() => props.ofoId, () => units.value.length], ([id]) => void loadPositions(id), { immediate: true });

const roleItems = computed(() => {
  const items = positions.value.map((p) => ({ value: p.name, label: p.name }));
  const current = props.role.trim();
  if (current && !items.some((i) => i.value === current)) {
    items.unshift({ value: current, label: `${current} — нет в справочнике ОФО` });
  }
  return items;
});

function onOfoChange(id: number | null) {
  if (id === props.ofoId) return;
  emit('update:ofoId', id);
  // Должность зависит от подразделения — при смене ОФО выбираем заново.
  emit('update:role', '');
}

const selectedRole = computed({
  get: () => props.role || undefined,
  set: (v: string | undefined) => emit('update:role', v ?? ''),
});

const ofoHelpText = computed(() => (ofoError.value ? String(ofoError.value) : props.ofoHelp));
</script>

<template>
  <UFormField label="ОФО" name="ofoId" :required="required" :help="ofoHelpText">
    <OfoSelect :model-value="ofoId" @update:model-value="onOfoChange" />
  </UFormField>

  <UFormField label="Должность" name="role" :required="required">
    <USelectMenu
      v-model="selectedRole"
      :items="roleItems"
      value-key="value"
      label-key="label"
      :placeholder="ofoId == null ? 'Сначала выберите ОФО' : 'Выберите должность'"
      :size="size"
      color="neutral"
      class="w-full"
      :disabled="disabled || ofoId == null || positionsLoading"
      :loading="positionsLoading"
      :content="{ align: 'start', sideOffset: 8 }"
    />
  </UFormField>
</template>
