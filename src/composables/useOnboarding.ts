import { avatarUrlFromFilename, DEFAULT_PROFILE_AVATAR_FILENAME } from '../constants/profileAvatars';

export type ProfileSnapshot = {
  id: number;
  firstname: string;
  surname: string;
  lastname: string;
  ofo: string | number;
  role: string;
  avatar_url: string;
  phone?: string;
  email?: string;
};

const SESSION_KEY_PREFIX = 'onboarding-complete:v1:';

export function onboardingSessionKey(userId: number): string {
  return `${SESSION_KEY_PREFIX}${userId}`;
}

export function isOfoUnset(ofo: unknown): boolean {
  const s = String(ofo ?? '').trim();
  return !s || s === '-1';
}

export function isProfileIncomplete(profile: Pick<ProfileSnapshot, 'ofo' | 'role' | 'avatar_url'>): boolean {
  if (isOfoUnset(profile.ofo)) return true;
  if (!String(profile.role ?? '').trim()) return true;
  if (!String(profile.avatar_url ?? '').trim()) return true;
  return false;
}

export function markOnboardingComplete(userId: number): void {
  if (typeof window === 'undefined') return;
  sessionStorage.setItem(onboardingSessionKey(userId), '1');
}

export function isOnboardingMarkedComplete(userId: number): boolean {
  if (typeof window === 'undefined') return false;
  return sessionStorage.getItem(onboardingSessionKey(userId)) === '1';
}

export async function fetchProfileSnapshot(userId: number): Promise<ProfileSnapshot | null> {
  try {
    const res = await fetch(`/api/profile.php?id=${userId}`);
    const data = await res.json();
    if (!data.success || !data.data) return null;
    const p = data.data;
    return {
      id: Number(p.id),
      firstname: p.firstname ?? '',
      surname: p.surname ?? '',
      lastname: p.lastname ?? '',
      ofo: p.ofo ?? '',
      role: p.role ?? '',
      avatar_url: p.avatar_url ?? '',
      phone: p.phone ?? '',
      email: p.email ?? '',
    };
  } catch {
    return null;
  }
}

export async function userNeedsOnboarding(userId: number): Promise<boolean> {
  const profile = await fetchProfileSnapshot(userId);
  if (!profile) return true;
  const incomplete = isProfileIncomplete(profile);
  if (!incomplete) {
    markOnboardingComplete(userId);
    return false;
  }
  return true;
}

export type SaveOnboardingPayload = {
  id: number;
  ofo: string;
  role: string;
  avatar_url: string;
  firstname?: string;
  surname?: string;
  lastname?: string;
  phone?: string;
  email?: string;
};

export async function saveOnboardingProfile(payload: SaveOnboardingPayload): Promise<boolean> {
  try {
    const res = await fetch('/api/profile.php', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    const data = await res.json();
    return Boolean(data.success);
  } catch {
    return false;
  }
}

export function defaultAvatarUrl(): string {
  return avatarUrlFromFilename(DEFAULT_PROFILE_AVATAR_FILENAME);
}
