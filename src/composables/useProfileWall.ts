import { computed, ref } from 'vue';

export type WallPost = {
  id: string;
  userId: number;
  authorName: string;
  authorAvatar: string;
  content: string;
  createdAt: string;
};

const STORAGE_KEY = 'profile-wall-posts:v2';

function normalizePost(raw: unknown): WallPost | null {
  if (!raw || typeof raw !== 'object') return null;
  const r = raw as Record<string, unknown>;
  if (!r.id || !r.createdAt) return null;

  let content = String(r.content ?? '');
  const images = Array.isArray(r.images) ? r.images.filter((i) => typeof i === 'string') as string[] : [];

  if (images.length && !content.includes('<img')) {
    const imgs = images.map((src) => `<p><img src="${src}" alt="" /></p>`).join('');
    content = content && !content.includes('<') ? `<p>${content}</p>${imgs}` : content ? `${content}${imgs}` : imgs;
  } else if (content && !content.includes('<')) {
    content = `<p>${content}</p>`;
  }

  return {
    id: String(r.id),
    userId: Number(r.userId) || 0,
    authorName: String(r.authorName ?? 'Сотрудник'),
    authorAvatar: String(r.authorAvatar ?? ''),
    content,
    createdAt: String(r.createdAt),
  };
}

function readPosts(): WallPost[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      const legacy = localStorage.getItem('profile-wall-posts:v1');
      if (legacy) {
        const parsed = JSON.parse(legacy);
        const migrated = (Array.isArray(parsed) ? parsed : [])
          .map(normalizePost)
          .filter((p): p is WallPost => p != null);
        writePosts(migrated);
        localStorage.removeItem('profile-wall-posts:v1');
        return migrated;
      }
      return [];
    }
    const parsed = JSON.parse(raw);
    return (Array.isArray(parsed) ? parsed : [])
      .map(normalizePost)
      .filter((p): p is WallPost => p != null);
  } catch {
    return [];
  }
}

function writePosts(posts: WallPost[]) {
  if (typeof window === 'undefined') return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(posts));
}

function newId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

const posts = ref<WallPost[]>(readPosts());

export function useProfileWall() {
  const loaded = ref(false);

  function ensureLoaded() {
    if (loaded.value) return;
    posts.value = readPosts();
    loaded.value = true;
  }

  function reload() {
    posts.value = readPosts();
    loaded.value = true;
  }

  function createPost(payload: {
    userId: number;
    authorName: string;
    authorAvatar: string;
    content: string;
  }) {
    const post: WallPost = {
      id: newId(),
      userId: payload.userId,
      authorName: payload.authorName,
      authorAvatar: payload.authorAvatar,
      content: payload.content.trim(),
      createdAt: new Date().toISOString(),
    };
    posts.value = [post, ...posts.value];
    writePosts(posts.value);
    return post;
  }

  function deletePost(id: string) {
    posts.value = posts.value.filter((p) => p.id !== id);
    writePosts(posts.value);
  }

  function updatePost(id: string, payload: { content: string }) {
    const content = payload.content.trim();
    posts.value = posts.value.map((p) =>
      p.id === id ? { ...p, content } : p,
    );
    writePosts(posts.value);
  }

  const sortedPosts = computed(() =>
    [...posts.value].sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
  );

  return {
    posts,
    sortedPosts,
    ensureLoaded,
    reload,
    createPost,
    deletePost,
    updatePost,
  };
}

export function formatWallDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffMin = Math.floor(diffMs / 60_000);
  if (diffMin < 1) return 'только что';
  if (diffMin < 60) return `${diffMin} мин. назад`;
  const diffH = Math.floor(diffMin / 60);
  if (diffH < 24) return `${diffH} ч. назад`;
  return d.toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function wallPostPlainText(html: string): string {
  return String(html || '')
    .replace(/<[^>]*>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function isWallContentEmpty(html: string): boolean {
  const plain = wallPostPlainText(html);
  if (plain) return false;
  return !String(html || '').includes('<img');
}
