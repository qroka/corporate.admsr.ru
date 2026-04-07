import type { FormListItem, FormWithQuestions, Report, SubmitPayload, SubmitResult, UUID } from './types'

function getAuthUserId(): string | null {
  try {
    const raw = localStorage.getItem('auth-user')
    const u = raw ? JSON.parse(raw) : null
    const id = u?.id
    if (id === null || id === undefined) return null
    const s = String(id).trim()
    return s ? s : null
  } catch {
    return null
  }
}

async function jsonFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const uid = getAuthUserId()
  const res = await fetch(url, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(uid ? { 'X-User-Id': uid } : {}),
      ...(init?.headers ?? {}),
    },
  })
  const data = await res.json().catch(() => null)
  if (!res.ok || !data?.success) {
    const msg = data?.error || data?.message || `HTTP ${res.status}`
    throw new Error(msg)
  }
  return data.data as T
}

export async function apiCreateForm(payload: any) {
  return jsonFetch<{ id: UUID }>(`/api/forms.php`, { method: 'POST', body: JSON.stringify(payload) })
}

export async function apiGetForm(id: UUID) {
  return jsonFetch<FormWithQuestions>(`/api/forms.php?id=${encodeURIComponent(id)}`, { method: 'GET' })
}

export async function apiUpdateForm(id: UUID, payload: any) {
  return jsonFetch<null>(`/api/forms.php?id=${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(payload) })
}

export async function apiPublishForm(id: UUID) {
  return jsonFetch<null>(`/api/forms_publish.php?id=${encodeURIComponent(id)}`, { method: 'POST' })
}

export async function apiSubmitForm(id: UUID, payload: SubmitPayload) {
  return jsonFetch<SubmitResult>(`/api/forms_submit.php?id=${encodeURIComponent(id)}`, { method: 'POST', body: JSON.stringify(payload) })
}

export async function apiGetReport(id: UUID, params: Record<string, string | number | undefined>) {
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined) continue
    qs.set(k, String(v))
  }
  const suffix = qs.toString() ? `&${qs.toString()}` : ''
  return jsonFetch<Report>(`/api/forms_report.php?id=${encodeURIComponent(id)}${suffix}`, { method: 'GET' })
}

export async function apiListForms(params: Record<string, string | number | undefined>) {
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined) continue
    qs.set(k, String(v))
  }
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return jsonFetch<FormListItem[]>(`/api/forms_list.php${suffix}`, { method: 'GET' })
}

export async function apiArchiveForm(id: UUID, nextStatus: 'archived' | 'draft') {
  return jsonFetch<null>(`/api/forms_archive.php?id=${encodeURIComponent(id)}`, {
    method: 'POST',
    body: JSON.stringify({ status: nextStatus }),
  })
}

export async function apiDeleteForm(id: UUID) {
  return jsonFetch<null>(`/api/forms_delete.php?id=${encodeURIComponent(id)}`, { method: 'DELETE' })
}

