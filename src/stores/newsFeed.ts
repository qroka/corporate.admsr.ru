import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { NewsRecord } from '../composables/useNewsData';

export type NewsFeedTab = 'feed' | 'ofo';

/**
 * In-memory SPA store for home news feed: loaded pages + scroll position.
 * Not persisted to localStorage.
 */
export const useNewsFeedStore = defineStore('newsFeed', () => {
  const items = ref<NewsRecord[]>([]);
  const cursor = ref<string | null>(null);
  const hasMore = ref(true);
  const scrollTop = ref(0);
  const activeTab = ref<NewsFeedTab>('feed');
  /** Category filter used for the last successful load (OFО tab). */
  const categoryFilter = ref<string | null>(null);
  const hydrated = ref(false);

  function reset() {
    items.value = [];
    cursor.value = null;
    hasMore.value = true;
    scrollTop.value = 0;
    categoryFilter.value = null;
    hydrated.value = false;
  }

  function setScrollTop(y: number) {
    scrollTop.value = Math.max(0, Number(y) || 0);
  }

  return {
    items,
    cursor,
    hasMore,
    scrollTop,
    activeTab,
    categoryFilter,
    hydrated,
    reset,
    setScrollTop,
  };
});
