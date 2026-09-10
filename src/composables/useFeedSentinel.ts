import { nextTick, onMounted, onUnmounted, watch, type Ref } from 'vue';

/**
 * IntersectionObserver helper: calls onIntersect when sentinel enters root (+ rootMargin).
 */
export function useFeedSentinel(opts: {
  root: Ref<HTMLElement | null>;
  sentinel: Ref<HTMLElement | null>;
  enabled: Ref<boolean>;
  onIntersect: () => void;
  rootMargin?: string;
}) {
  let observer: IntersectionObserver | null = null;

  function disconnect() {
    observer?.disconnect();
    observer = null;
  }

  function connect() {
    disconnect();
    const root = opts.root.value;
    const target = opts.sentinel.value;
    if (!root || !target || typeof IntersectionObserver === 'undefined') return;

    observer = new IntersectionObserver(
      (entries) => {
        const hit = entries.some((e) => e.isIntersecting);
        if (!hit) return;
        if (!opts.enabled.value) return;
        opts.onIntersect();
      },
      {
        root,
        rootMargin: opts.rootMargin ?? '600px 0px',
        threshold: 0,
      },
    );
    observer.observe(target);
  }

  onMounted(() => {
    void nextTick(() => connect());
  });

  onUnmounted(() => {
    disconnect();
  });

  watch(
    [() => opts.root.value, () => opts.sentinel.value, opts.enabled],
    () => {
      void nextTick(() => connect());
    },
  );

  return { connect, disconnect };
}
