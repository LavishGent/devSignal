// Thin typed wrapper over fetch. All requests are same-origin (Vite proxies
// /api in development); the session rides an HttpOnly cookie, so no tokens
// are ever handled in JavaScript.

export class ApiError extends Error {
  status: number
  fields: Record<string, string>

  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message)
    this.status = status
    this.fields = fields ?? {}
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: 'same-origin',
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
    ...init,
  })

  if (!res.ok) {
    let message = res.statusText
    let fields: Record<string, string> | undefined
    try {
      const body = await res.json()
      if (typeof body?.error === 'string') message = body.error
      if (body?.fields) fields = body.fields
    } catch {
      // non-JSON error body; keep the status text
    }
    throw new ApiError(res.status, message, fields)
  }
  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body === undefined ? undefined : JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}

// buildQuery renders non-empty params as a query string.
export function buildQuery(params: Record<string, string>): string {
  const entries = Object.entries(params).filter(([, v]) => v !== '')
  if (entries.length === 0) return ''
  return '?' + new URLSearchParams(Object.fromEntries(entries)).toString()
}
