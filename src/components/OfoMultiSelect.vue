<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useOfoTree, type OfoUnit } from '../composables/useOfoTree';

const model = defineModel<number[]>({ default: () => [] });

const { categories, units, loading, error, ensureLoaded, rootUnitsOf, childrenOf, hasChildren, unitById } = useOfoTree();
ensureLoaded();

const open = ref(false);
const expanded = reactive<Set<number>>(new Set());
const query = ref('');
const isSearching = computed(() => query.value.trim().length > 0);

const selectedSet = computed(() => new Set(model.value));
const count = computed(() => model.value.length);

const sortedCategories = computed(() =>
  [...categories.value].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id),
);

// ── Каскадный выбор ───────────────────────────────────────────────────────────
function withDescendants(id: number): number[] {
  const out = [id];
  for (const ch of childrenOf(id)) out.push(...withDescendants(ch.id));
  return out;
}
function isChecked(id: number) {
  return selectedSet.value.has(id);
}
function hasSelectedDescendant(id: number): boolean {
  return childrenOf(id).some((ch) => isChecked(ch.id) || hasSelectedDescendant(ch.id));
}
function isIndeterminate(id: number) {
  return !isChecked(id) && hasSelectedDescendant(id);
}
function toggle(unit: OfoUnit) {
  const set = new Set(model.value);
  const ids = withDescendants(unit.id);
  if (set.has(unit.id)) ids.forEach((i) => set.delete(i));
  else ids.forEach((i) => set.add(i));
  model.value = [...set];
}
function clearAll() {
  model.value = [];
}

// ── Поиск ─────────────────────────────────────────────────────────────────────
const matches = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return [];
  return units.value
    .filter((u) => u.name.toLowerCase().includes(q))
    .sort((a, b) => a.name.localeCompare(b.name, 'ru'));
});

// ── Дерево ────────────────────────────────────────────────────────────────────
function flatten(categoryId: number): { unit: OfoUnit; depth: number }[] {
  const out: { unit: OfoUnit; depth: number }[] = [];
  const walk = (unit: OfoUnit, depth: number) => {
    out.push({ unit, depth });
    for (const ch of childrenOf(unit.id)) walk(ch, depth + 1);
  };
  for (const root of rootUnitsOf(categoryId)) walk(root, 0);
  return out;
}
function isVisible(unit: OfoUnit): boolean {
  let parentId = unit.parent_id;
  while (parentId != null) {
    if (!expanded.has(parentId)) return false;
    parentId = unitById.value.get(parentId)?.parent_id ?? null;
  }
  return true;
}
function toggleExpand(id: number) {
  if (expanded.has(id)) expanded.delete(id);
  else expanded.add(id);
}
</script>

<template>
  <UPopover v-model:open="open" :content="{ align: 'start', sideOffset: 6 }" :ui="{ content: 'w-(--reka-popper-anchor-width) p-0' }">
    <UButton color="neutral" variant="outline" size="xl" trailing-icon="i-lucide-chevron-down" class="w-full justify-between">
      <span class="truncate" :class="count ? '' : 'text-dimmed'">
        {{ count ? `Выбрано ОФО: ${count}` : 'Выбрать ОФО' }}
      </span>
    </UButton>

    <template #content>
      <div class="p-2">
        <div v-if="loading" class="text-sm text-muted px-1 py-2">Загрузка ОФО…</div>
        <div v-else-if="error" class="text-sm text-error px-1 py-2">{{ error }}</div>

        <div v-else class="flex flex-col gap-2">
          <div class="flex items-center gap-2">
            <UInput v-model="query" icon="i-lucide-search" placeholder="Поиск…" size="md" class="flex-1" />
            <UButton v-if="count" color="neutral" variant="ghost" size="sm" icon="i-lucide-x" @click="clearAll">Сброс</UButton>
          </div>

          <div class="flex flex-col gap-0.5 max-h-72 overflow-y-auto scrollbar-hide px-0.5 py-0.5">
            <!-- Поиск -->
            <template v-if="isSearching">
              <p v-if="!matches.length" class="text-sm text-muted px-2 py-2">Ничего не найдено</p>
              <button
                v-for="u in matches"
                :key="`m-${u.id}`"
                type="button"
                class="flex items-center gap-2 w-full text-left rounded-lg px-2 py-1.5 hover:bg-elevated"
                @click="toggle(u)"
              >
                <UCheckbox :model-value="isChecked(u.id)" :indeterminate="isIndeterminate(u.id)" @update:model-value="toggle(u)" @click.stop />
                <span class="text-sm truncate">{{ u.name }}</span>
              </button>
            </template>

            <!-- Дерево -->
            <template v-else>
              <template v-for="cat in sortedCategories" :key="`c-${cat.id}`">
                <div class="px-2 pt-2 pb-1 text-xs font-medium uppercase tracking-wide text-dimmed select-none">
                  {{ cat.name }}
                </div>
                <template v-for="row in flatten(cat.id)" :key="row.unit.id">
                  <div
                    v-if="isVisible(row.unit)"
                    class="flex items-center gap-1 w-full rounded-lg hover:bg-elevated"
                    :style="{ paddingLeft: `${row.depth * 18}px` }"
                  >
                    <button
                      v-if="hasChildren(row.unit.id)"
                      type="button"
                      class="shrink-0 p-1.5"
                      @click.stop="toggleExpand(row.unit.id)"
                    >
                      <UIcon
                        :name="expanded.has(row.unit.id) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
                        class="size-4 text-muted"
                      />
                    </button>
                    <span v-else class="inline-block size-4 shrink-0 ml-1.5" aria-hidden="true" />
                    <label class="flex items-center gap-2 flex-1 min-w-0 py-1.5 pr-2 cursor-pointer">
                      <UCheckbox :model-value="isChecked(row.unit.id)" :indeterminate="isIndeterminate(row.unit.id)" @update:model-value="toggle(row.unit)" />
                      <span class="text-sm truncate">{{ row.unit.name }}</span>
                    </label>
                  </div>
                </template>
              </template>
            </template>
          </div>
        </div>
      </div>
    </template>
  </UPopover>
</template>
