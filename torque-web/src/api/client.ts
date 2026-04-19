/**
 * Torque HTTP Client
 *
 * Wrapper over fetch with CSRF protection, request tracing,
 * automatic token refresh with mutex, and typed error handling.
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export class AppError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.name = 'AppError'
    this.code = code
    this.status = status
  }
}

export interface ApiResponse<T> {
  data: T
  meta?: Record<string, unknown>
}

type HttpMethod = 'GET' | 'POST' | 'PATCH' | 'DELETE'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const API_BASE = (import.meta.env.VITE_API_URL as string | undefined) ?? ''

function readCsrfToken(): string {
  const match = document.cookie.split('; ').find((row) => row.startsWith('__torque_csrf='))
  return match ? (match.split('=')[1] ?? '') : ''
}

function generateRequestId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  // Fallback — 128-bit hex
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}

function looksLikeAppErrorPayload(
  value: unknown
): value is { code: string; message: string; status: number } {
  return (
    typeof value === 'object' &&
    value !== null &&
    'code' in value &&
    'message' in value &&
    'status' in value
  )
}

// ---------------------------------------------------------------------------
// Refresh mutex — prevents concurrent refresh calls
// ---------------------------------------------------------------------------

let refreshPromise: Promise<boolean> | null = null

async function refreshToken(): Promise<boolean> {
  if (refreshPromise) return refreshPromise

  refreshPromise = (async () => {
    try {
      // Refresh cookie lives under /api/v1/auth; the same path is required here.
      const res = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'X-CSRF-Token': readCsrfToken(),
          'X-Request-ID': generateRequestId(),
        },
      })
      return res.ok
    } catch {
      return false
    } finally {
      refreshPromise = null
    }
  })()

  return refreshPromise
}

// ---------------------------------------------------------------------------
// Core request
// ---------------------------------------------------------------------------

async function request<T>(
  method: HttpMethod,
  path: string,
  body?: unknown,
  retriedAfterRefresh = false
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
    'X-Request-ID': generateRequestId(),
  }

  // CSRF token for mutating methods
  if (method !== 'GET') {
    headers['X-CSRF-Token'] = readCsrfToken()
  }

  const init: RequestInit = {
    method,
    credentials: 'include',
    headers,
  }

  if (body !== undefined && method !== 'GET') {
    init.body = JSON.stringify(body)
  }

  const res = await fetch(`${API_BASE}${path}`, init)

  // --- 401: attempt refresh once ---
  if (res.status === 401 && !retriedAfterRefresh) {
    const refreshed = await refreshToken()
    if (refreshed) {
      return request<T>(method, path, body, true)
    }
    window.dispatchEvent(new CustomEvent('auth:logout'))
    throw new AppError('AUTH_EXPIRED', 'Session expired', 401)
  }

  // --- 429: rate limit ---
  if (res.status === 429) {
    const retryAfter = res.headers.get('Retry-After')
    throw new AppError(
      'RATE_LIMITED',
      `Rate limited${retryAfter ? `. Retry after ${retryAfter}s` : ''}`,
      429
    )
  }

  // --- Other errors ---
  if (!res.ok) {
    let errorBody: unknown
    try {
      errorBody = await res.json()
    } catch {
      errorBody = null
    }

    if (looksLikeAppErrorPayload(errorBody)) {
      throw new AppError(errorBody.code, errorBody.message, errorBody.status)
    }

    throw new AppError(
      'REQUEST_FAILED',
      typeof errorBody === 'object' && errorBody !== null && 'message' in errorBody
        ? String((errorBody as Record<string, unknown>).message)
        : `Request failed with status ${res.status}`,
      res.status
    )
  }

  // 204 No Content
  if (res.status === 204) {
    return undefined as T
  }

  const json: unknown = await res.json()
  return json as T
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

export function get<T>(path: string): Promise<T> {
  return request<T>('GET', path)
}

export function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('POST', path, body)
}

export function patch<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('PATCH', path, body)
}

export function del<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('DELETE', path, body)
}
