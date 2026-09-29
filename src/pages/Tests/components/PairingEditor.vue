<script setup lang="ts">
import { computed } from 'vue';
import {
  addPair,
  createItem,
  createOption,
  pairingCorrect,
  removeCategory,
  removePair,
  unusedTargets,
  type QOption,
  type Question,
} from '../questionTypes';

/**
 * Редактор вопросов «Соответствие» и «Классификация».
 *  match    — пары «понятие → определение» (+ лишние определения без пары);
 *  classify — категории и утверждения; у каждого утверждения выбирается его категория.
 * Правильный ответ (ключ) здесь же: у match это сама пара, у classify — выбор категории
 * (только в тесте; в опросе ключ не нужен).
 */
const props = defineProps<{ q: Question; kind: 'test' | 'survey' | 'poll' }>();

const map = computed(() => pairingCorrect(props.q));
const isTest = computed(() => props.kind === 'test');

function targetOf(itemId: string): QOption | undefined {
  return props.q.options.find((o) => o.id === map.value[itemId]);
}

const categoryItems = computed(() =>
  props.q.options.map((o, i) => ({ label: o.text.trim() || `Категория ${i + 1}`, value: o.id })),
);

function setKey(itemId: string, targetId: string | undefined) {
  const next = { ...map.value };
  if (targetId) next[itemId] = targetId;
  else delete next[itemId];
  props.q.correct = next;
}

// match
const extraTargets = computed(() => unusedTargets(props.q));
function addExtraDefinition() {
  props.q.options.push(createOption());
}
function removeExtraDefinition(id: string) {
  props.q.options = props.q.options.filter((o) => o.id !== id);
}

// classify
function addCategory() {
  props.q.options.push(createOption());
}
function addStatement() {
  props.q.items.push(createItem());
}
function removeStatement(index: number) {
  const [it] = props.q.items.splice(index, 1);
  if (!it) return;
  const next = { ...map.value };
  delete next[it.id];
  props.q.correct = next;
}
</script>

<template>
  <!-- ═══ Соответствие ═══ -->
  <div v-if="q.type === 'match'" class="flex flex-col gap-3">
    <p class="text-xs text-dimmed">
      Слева — понятие, справа — его определение. Сотруднику определения покажутся перемешанными.
    </p>
    <div v-for="(it, i) in q.items" :key="it.id" class="flex flex-col sm:flex-row sm:items-center gap-2">
      <UInput v-model="it.text" size="md" class="sm:w-2/5" :placeholder="`Понятие ${i + 1}`" />
      <UIcon name="i-lucide-arrow-right" class="hidden sm:block size-4 shrink-0 text-dimmed" aria-hidden="true" />
      <UInput
        v-if="targetOf(it.id)"
        v-model="targetOf(it.id)!.text"
        size="md"
        class="flex-1"
        :color="isTest ? 'success' : undefined"
        :placeholder="`Определение ${i + 1}`"
      />
      <UTooltip text="Удалить пару">
        <UButton
          color="error"
          variant="ghost"
          size="sm"
          icon="i-lucide-x"
          :disabled="q.items.length <= 2"
          aria-label="Удалить пару"
          @click="removePair(q, i)"
        />
      </UTooltip>
    </div>
    <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-plus" class="w-fit" @click="addPair(q)">
      Добавить пару
    </UButton>

    <div class="flex flex-col gap-2 pt-1">
      <p class="text-xs text-dimmed">Лишние определения — без пары, для усложнения (необязательно).</p>
      <div v-for="o in extraTargets" :key="o.id" class="flex items-center gap-2">
        <UIcon name="i-lucide-corner-down-right" class="size-4 shrink-0 text-dimmed" aria-hidden="true" />
        <UInput v-model="o.text" size="md" class="flex-1" placeholder="Лишнее определение" />
        <UTooltip text="Удалить">
          <UButton
            color="error"
            variant="ghost"
            size="sm"
            icon="i-lucide-x"
            aria-label="Удалить лишнее определение"
            @click="removeExtraDefinition(o.id)"
          />
        </UTooltip>
      </div>
      <UButton color="neutral" variant="ghost" size="sm" icon="i-lucide-plus" class="w-fit" @click="addExtraDefinition">
        Добавить лишнее определение
      </UButton>
    </div>
  </div>

  <!-- ═══ Классификация ═══ -->
  <div v-else class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <p class="text-sm font-medium text-highlighted">Категории</p>
      <div v-for="(o, oi) in q.options" :key="o.id" class="flex items-center gap-2">
        <UIcon name="i-lucide-folder" class="size-4 shrink-0 text-dimmed" aria-hidden="true" />
        <UInput v-model="o.text" size="md" class="flex-1" :placeholder="`Категория ${oi + 1}`" />
        <UTooltip text="Удалить категорию">
          <UButton
            color="error"
            variant="ghost"
            size="sm"
            icon="i-lucide-x"
            :disabled="q.options.length <= 2"
            aria-label="Удалить категорию"
            @click="removeCategory(q, oi)"
          />
        </UTooltip>
      </div>
      <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-plus" class="w-fit" @click="addCategory">
        Добавить категорию
      </UButton>
    </div>

    <div class="flex flex-col gap-2">
      <p class="text-sm font-medium text-highlighted">
        Утверждения<span v-if="isTest" class="font-normal text-muted"> — и категория, к которой каждое относится</span>
      </p>
      <div v-for="(it, i) in q.items" :key="it.id" class="flex flex-col sm:flex-row sm:items-center gap-2">
        <UInput v-model="it.text" size="md" class="flex-1" :placeholder="`Утверждение ${i + 1}`" />
        <USelect
          v-if="isTest"
          :model-value="map[it.id]"
          :items="categoryItems"
          value-key="value"
          size="md"
          class="sm:w-56"
          placeholder="Категория…"
          :color="map[it.id] ? 'success' : undefined"
          :content="{ align: 'start', sideOffset: 8 }"
          @update:model-value="(v: string) => setKey(it.id, v)"
        />
        <UTooltip text="Удалить утверждение">
          <UButton
            color="error"
            variant="ghost"
            size="sm"
            icon="i-lucide-x"
            :disabled="q.items.length <= 1"
            aria-label="Удалить утверждение"
            @click="removeStatement(i)"
          />
        </UTooltip>
      </div>
      <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-plus" class="w-fit" @click="addStatement">
        Добавить утверждение
      </UButton>
    </div>
  </div>
</template>
