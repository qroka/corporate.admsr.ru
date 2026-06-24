<script setup lang="ts">
import { computed, reactive } from 'vue';
import { useOfoTree, type OfoUnit } from '../composables/useOfoTree';

const { categories, units, loading, error, ensureLoaded, rootUnitsOf, childrenOf, hasChildren, unitById, pathLabel } = useOfoTree();
ensureLoaded();

const query = reactive({ q: '' });
const expanded = reactive<Set<number>>(new Set());

const sortedCategories = computed(() =>
  [...categories.value].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id),
);

const totals = computed(() => ({
  categories: categories.value.length,
  units: units.value.length,
}));

const isSearching = computed(() => query.q.trim().length > 0);
const matches = computed(() => {
  const q = query.q.trim().toLowerCase();
  if (!q) return [];
  return units.value
    .filter((u) => u.name.toLowerCase().includes(q))
    .sort((a, b) => a.name.localeCompare(b.name, 'ru'));
});
const categoryName = (id: number) => categories.value.find((c) => c.id === id)?.name ?? '';
const parentLabel = (u: OfoUnit) => (u.parent_id != null ? pathLabel(u.parent_id) : categoryName(u.category_id));

function flatten(categoryId: number): { unit: OfoUnit; depth: number }[] {
  const out: { unit: OfoUnit; depth: number }[] = [];
  const walk = (unit: OfoUnit, depth: number) => {
    out.push({ unit, depth });
    for (const ch of childrenOf(unit.id)) walk(ch, depth + 1);
  };
  for (const root of rootUnitsOf(categoryId)) walk(root, 0);
  return out;
}
/** Сумма участников по всему поддереву (узел + все дочерние). */
function subtreeUserCount(unit: OfoUnit): number {
  let sum = unit.user_count ?? 0;
  for (const ch of childrenOf(unit.id)) sum += subtreeUserCount(ch);
  return sum;
}

function isVisible(unit: OfoUnit): boolean {
  let parentId = unit.parent_id;
  while (parentId != null) {
    if (!expanded.has(parentId)) return false;
    parentId = unitById.value.get(parentId)?.parent_id ?? null;
  }
  return true;
}
function toggle(id: number) {
  if (expanded.has(id)) expanded.delete(id);
  else expanded.add(id);
}
function expandAll() {
  for (const u of units.value) if (hasChildren(u.id)) expanded.add(u.id);
}
function collapseAll() {
  expanded.clear();
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex flex-col sm:flex-row gap-3 sm:items-center">
      <UInput
        v-model="query.q"
        icon="i-lucide-search"
        size="lg"
        color="neutral"
        variant="outline"
        placeholder="Поиск по названию…"
        class="w-full sm:flex-1"
      />
      <div class="flex items-center gap-2">
        <UButton color="neutral" variant="outline" size="md" icon="i-lucide-unfold-more" @click="expandAll">Развернуть</UButton>
        <UButton color="neutral" variant="outline" size="md" icon="i-lucide-unfold-less" @click="collapseAll">Свернуть</UButton>
      </div>
    </div>

    <p class="text-xs text-muted px-1">
      Категорий: {{ totals.categories }} · подразделений: {{ totals.units }}
    </p>

    <div v-if="loading" class="text-sm text-muted py-4 px-2">Загрузка структуры…</div>
    <div v-else-if="error" class="text-sm text-error py-4 px-2">{{ error }}</div>

    <div v-else class="flex flex-col gap-1 rounded-xl ring-1 ring-default p-2 max-h-[60vh] overflow-y-auto scrollbar-hide">
      <!-- Поиск -->
      <template v-if="isSearching">
        <p v-if="!matches.length" class="text-sm text-muted px-2 py-2">Ничего не найдено</p>
        <div
          v-for="u in matches"
          :key="`m-${u.id}`"
          class="flex items-center justify-between gap-3 rounded-lg px-2 py-1.5 hover:bg-elevated"
        >
          <div class="min-w-0">
            <div class="text-sm text-default truncate">{{ u.name }}</div>
            <div class="text-xs text-dimmed truncate">{{ parentLabel(u) }}</div>
          </div>
          <div class="flex items-center gap-1.5 shrink-0">
            <UBadge v-if="hasChildren(u.id)" color="primary" variant="subtle" size="sm">{{ subtreeUserCount(u) }} всего</UBadge>
            <UBadge color="info" variant="subtle" size="sm">{{ u.user_count ?? 0 }} польз.</UBadge>
          </div>
        </div>
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
              class="flex items-center justify-between gap-3 rounded-lg px-2 py-1.5 hover:bg-elevated"
              :style="{ paddingLeft: `${8 + row.depth * 18}px` }"
            >
              <button
                type="button"
                class="flex items-center gap-1.5 min-w-0 text-left"
                @click="hasChildren(row.unit.id) && toggle(row.unit.id)"
              >
                <UIcon
                  v-if="hasChildren(row.unit.id)"
                  :name="expanded.has(row.unit.id) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
                  class="size-4 shrink-0 text-muted"
                />
                <span v-else class="inline-block size-4 shrink-0" aria-hidden="true" />
                <span class="text-sm text-default truncate">{{ row.unit.name }}</span>
              </button>
              <div class="flex items-center gap-1.5 shrink-0">
                <UBadge v-if="hasChildren(row.unit.id)" color="primary" variant="subtle" size="sm">{{ subtreeUserCount(row.unit) }} всего</UBadge>
                <UBadge color="info" variant="subtle" size="sm">{{ row.unit.user_count ?? 0 }} польз.</UBadge>
              </div>
            </div>
          </template>
        </template>
      </template>
    </div>
  </div>
</template>
