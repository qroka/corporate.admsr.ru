import { computed, onMounted, ref } from 'vue';

export type EventRecord = {
  id: number;
  title: string;
  description: string;
  date: string; // YYYY-MM-DD
  badge?: string;
  coverIndex?: number;
};

function isEventRecord(x: any): x is EventRecord {
  return x && typeof x === 'object' && typeof x.id === 'number' && typeof x.title === 'string';
}

const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedEvents = ref<EventRecord[]>([]);
const sharedLoaded = ref(false);
let sharedPromise: Promise<void> | null = null;

export function useEventsData() {
  const loading = sharedLoading;
  const error = sharedError;
  const events = sharedEvents;

  async function load() {
    if (sharedLoaded.value) return;
    if (sharedPromise) return sharedPromise;

    sharedLoading.value = true;
    sharedError.value = null;

    sharedPromise = (async () => {
      try {
        const res = await fetch('/data/events.json');
        if (!res.ok) throw new Error(`Не удалось загрузить events.json: ${res.status}`);
        const data = await res.json();
        if (!Array.isArray(data)) throw new Error('events.json должен быть массивом');
        sharedEvents.value = data
          .filter(isEventRecord)
          .map((e) => ({
            id: Number(e.id),
            title: String(e.title ?? '').trim(),
            description: String(e.description ?? '').trim(),
            date: String(e.date ?? '').trim(),
            badge: e.badge ? String(e.badge) : undefined,
            coverIndex: typeof e.coverIndex === 'number' ? e.coverIndex : undefined,
          }))
          .filter((e) => e.id && e.title && e.date);
        sharedLoaded.value = true;
      } catch (e) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки мероприятий';
        sharedEvents.value = [];
      } finally {
        sharedLoading.value = false;
        sharedPromise = null;
      }
    })();

    return sharedPromise;
  }

  function ensureLoaded() {
    void load();
  }

  const badges = computed(() => {
    const set = new Set<string>();
    for (const e of sharedEvents.value) if (e.badge) set.add(e.badge);
    return Array.from(set).sort((a, b) => a.localeCompare(b, 'ru'));
  });

  onMounted(() => {
    // no auto-load by default (lazy from pages)
  });

  return { loading, error, events, badges, load, ensureLoaded };
}

