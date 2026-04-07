import type { UUID } from './types'

export function getOrCreateSessionId(formId: UUID) {
  const key = `tests:session:${formId}`
  const existing = localStorage.getItem(key)
  if (existing && /^[0-9a-fA-F-]{36}$/.test(existing)) return existing
  const next = crypto.randomUUID()
  localStorage.setItem(key, next)
  return next
}

export function progressKey(formId: UUID, sessionId: UUID) {
  return `tests:progress:${formId}:${sessionId}`
}

