import { computed, onMounted, ref } from 'vue';

type OfoExportRow = {
  id: string | number;
  title: string;
  parent?: string | number;
  type?: string;
  sort_order?: string | number;
};

type OfficeSeatExportRow = {
  id: string | number;
  title: string;
  ofo: string | number;
  insurance: string | number;
  rating: string | number;
};

export type OfoStat = {
  id: string;
  title: string;
  parentId?: string;
  parentTitle?: string;
  seatsCount: number;
  insuranceCount: number;
  ratingCount: number;
};

export type OfficeSeatRow = {
  id: string;
  title: string;
  ofo: string;
};

function normalizeString(value: unknown) {
  return String(value ?? '').trim();
}

function extractTableData(exportJson: unknown, tableName: string) {
  if (!Array.isArray(exportJson)) return [];
  for (const item of exportJson) {
    if (item && typeof item === 'object') {
      const maybe = item as any;
      if (maybe.type === 'table' && maybe.name === tableName && Array.isArray(maybe.data)) {
        return maybe.data;
      }
    }
  }
  return [];
}

export function useOfoData() {
  // shared cache (single fetch/parse per session)
  const loading = sharedLoading;
  const error = sharedError;

  const ofoRows = sharedOfoRows;
  const officeSeats = sharedOfficeSeats;
  const stats = sharedStats;

  const ofoTitleById = computed(() => {
    const map: Record<string, string> = {};
    for (const r of ofoRows.value) {
      const id = normalizeString(r.id);
      if (!id) continue;
      map[id] = r.title;
    }
    return map;
  });

  const ofoStats = computed<OfoStat[]>(() => {
    return ofoRows.value
      .map((o) => {
        const id = normalizeString(o.id);
        const parentId = o.parent !== undefined && o.parent !== null ? normalizeString(o.parent) : undefined;
        const s = stats.value[id] ?? { seatsCount: 0, insuranceCount: 0, ratingCount: 0 };
        return {
          id,
          title: o.title,
          parentId: parentId || undefined,
          parentTitle: parentId ? ofoTitleById.value[parentId] : undefined,
          seatsCount: s.seatsCount,
          insuranceCount: s.insuranceCount,
          ratingCount: s.ratingCount,
        } satisfies OfoStat;
      })
      .sort((a, b) => a.title.localeCompare(b.title, 'ru'));
  });

  async function loadDirectory() {
    if (sharedDirectoryLoaded.value) return;
    if (sharedDirectoryPromise) return sharedDirectoryPromise;

    sharedLoading.value = true;
    sharedError.value = null;

    sharedDirectoryPromise = (async () => {
      try {
        const ofoExport = await fetch('/data/ofo.json').then((r) => {
          if (!r.ok) throw new Error(`Не удалось загрузить ofo.json: ${r.status}`);
          return r.json();
        });

        const ofoTable = extractTableData(ofoExport, 'ofo') as OfoExportRow[];

        sharedOfoRows.value = ofoTable.map((r) => ({
          id: normalizeString(r.id),
          title: r.title,
          parent: r.parent !== undefined ? normalizeString(r.parent) : undefined,
          type: r.type,
          sort_order: r.sort_order,
        }));

        sharedDirectoryLoaded.value = true;
      } catch (e) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки данных OFO';
      } finally {
        sharedLoading.value = false;
        sharedDirectoryPromise = null;
      }
    })();

    return sharedDirectoryPromise;
  }

  async function loadSeats() {
    if (sharedSeatsLoaded.value) return;
    if (sharedSeatsPromise) return sharedSeatsPromise;

    // Ensure directory is present for parentTitle/depth calculations.
    await loadDirectory();

    sharedLoading.value = true;
    sharedError.value = null;

    sharedSeatsPromise = (async () => {
      try {
        const seatsExport = await fetch('/data/office_seats.json').then((r) => {
          if (!r.ok) throw new Error(`Не удалось загрузить office_seats.json: ${r.status}`);
          return r.json();
        });

        const seatsTable = extractTableData(seatsExport, 'office_seats') as OfficeSeatExportRow[];

        sharedOfficeSeats.value = seatsTable.map((r) => ({
          id: normalizeString(r.id),
          title: normalizeString(r.title),
          ofo: normalizeString(r.ofo),
        }));

        const nextStats: Record<string, { seatsCount: number; insuranceCount: number; ratingCount: number }> = {};
        for (const row of seatsTable) {
          const ofoId = normalizeString(row.ofo);
          if (!ofoId) continue;

          const insurance = Number(row.insurance);
          const rating = Number(row.rating);

          if (!nextStats[ofoId]) {
            nextStats[ofoId] = { seatsCount: 0, insuranceCount: 0, ratingCount: 0 };
          }

          nextStats[ofoId].seatsCount += 1;
          if (insurance > 0) nextStats[ofoId].insuranceCount += 1;
          if (rating > 0) nextStats[ofoId].ratingCount += 1;
        }

        sharedStats.value = nextStats;
        sharedSeatsLoaded.value = true;
      } catch (e) {
        sharedError.value = e instanceof Error ? e.message : 'Ошибка загрузки данных OFO';
      } finally {
        sharedLoading.value = false;
        sharedSeatsPromise = null;
      }
    })();

    return sharedSeatsPromise;
  }

  function ensureDirectoryLoaded() {
    void loadDirectory();
  }

  function ensureSeatsLoaded() {
    void loadSeats();
  }

  onMounted(() => {
    // no auto-load by default; caller can call ensureLoaded()
  });

  return {
    loading,
    error,
    ofoRows,
    ofoStats,
    officeSeats,
    loadDirectory,
    loadSeats,
    ensureDirectoryLoaded,
    ensureSeatsLoaded,
  };
}

// module-scope shared state (keeps data between composable calls)
const sharedLoading = ref(false);
const sharedError = ref<string | null>(null);
const sharedOfoRows = ref<OfoExportRow[]>([]);
const sharedOfficeSeats = ref<OfficeSeatRow[]>([]);
const sharedStats = ref<Record<string, { seatsCount: number; insuranceCount: number; ratingCount: number }>>({});
const sharedDirectoryLoaded = ref(false);
const sharedSeatsLoaded = ref(false);
let sharedDirectoryPromise: Promise<void> | null = null;
let sharedSeatsPromise: Promise<void> | null = null;

