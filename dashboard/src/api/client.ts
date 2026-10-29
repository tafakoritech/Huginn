import { config } from '@/config'

export type ApiErrorShape = {
  code: string
  message: string
  details?: Record<string, unknown>
}

export class ApiError extends Error {
  status: number
  code: string
  details?: Record<string, unknown>

  constructor(status: number, code: string, message: string, details?: Record<string, unknown>) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

export class NetworkError extends Error {
  constructor(message = 'Network failure') {
    super(message)
    this.name = 'NetworkError'
  }
}

type TokenStore = {
  getAccess: () => string | null
  getRefresh: () => string | null
  setTokens: (access: string, refresh: string) => void
  clear: () => void
}

const tokenStore: TokenStore = {
  getAccess: () => localStorage.getItem(config.storageKeys.accessToken),
  getRefresh: () => localStorage.getItem(config.storageKeys.refreshToken),
  setTokens: (access, refresh) => {
    localStorage.setItem(config.storageKeys.accessToken, access)
    localStorage.setItem(config.storageKeys.refreshToken, refresh)
  },
  clear: () => {
    localStorage.removeItem(config.storageKeys.accessToken)
    localStorage.removeItem(config.storageKeys.refreshToken)
    localStorage.removeItem(config.storageKeys.user)
  },
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let onUnauthorized: ((reason: string) => void) | null = null
export function setOnUnauthorized(fn: ((reason: string) => void) | null) {
  onUnauthorized = fn
}

export function setTokensForClient(access: string, refresh: string) {
  tokenStore.setTokens(access, refresh)
}

// --- Refresh flow with a shared promise to prevent refresh storms ---
let refreshPromise: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  const refresh = tokenStore.getRefresh()
  if (!refresh) return null
  try {
    const res = await fetch(`${config.apiBaseUrl}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Refresh-Token': refresh },
    })
    if (!res.ok) {
      tokenStore.clear()
      return null
    }
    const data = await res.json()
    const access = data?.access_token ?? data?.tokens?.access_token
    const newRefresh = data?.refresh_token ?? data?.tokens?.refresh_token ?? refresh
    if (!access) {
      tokenStore.clear()
      return null
    }
    tokenStore.setTokens(access, newRefresh)
    return access
  } catch {
    tokenStore.clear()
    return null
  }
}

function getRefreshToken(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = refreshAccessToken().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

async function performRequest(
  method: string,
  path: string,
  options: {
    body?: unknown
    headers?: Record<string, string>
    withAuth?: boolean
    isFormData?: boolean
    root?: boolean
    retried?: boolean
  } = {},
): Promise<Response> {
  const { body, headers = {}, withAuth = true, isFormData = false, root = false, retried = false } = options
  const base = root ? new URL(config.apiBaseUrl).origin : config.apiBaseUrl
  const url = path.startsWith('http') ? path : `${base}${path}`

  const mergedHeaders: Record<string, string> = { ...headers }
  if (!isFormData) {
    mergedHeaders['Content-Type'] = 'application/json'
  }
  const access = tokenStore.getAccess()
  if (withAuth && access) {
    mergedHeaders['Authorization'] = `Bearer ${access}`
  }

  let response: Response
  try {
    response = await fetch(url, {
      method,
      headers: mergedHeaders,
      body: body === undefined ? undefined : isFormData ? (body as FormData) : JSON.stringify(body),
    })
  } catch {
    throw new NetworkError()
  }

  if (response.status === 401 && withAuth && !retried) {
    const newAccess = await getRefreshToken()
    if (newAccess) {
      return performRequest(method, path, { ...options, retried: true })
    }
    onUnauthorized?.('session_expired')
    throw new ApiError(401, 'UNAUTHORIZED', 'Session expired')
  }

  return response
}

async function parseBody(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

async function handle<T>(response: Response): Promise<T> {
  const data = await parseBody(response)
  if (!response.ok) {
    const err = (data as { error?: ApiErrorShape })?.error
    if (err) {
      throw new ApiError(response.status, err.code, err.message, err.details)
    }
    throw new ApiError(response.status, 'HTTP_ERROR', `Request failed with status ${response.status}`)
  }
  return data as T
}

export async function apiGet<T>(path: string, opts?: { withAuth?: boolean; root?: boolean }): Promise<T> {
  const res = await performRequest('GET', path, opts)
  return handle<T>(res)
}

export async function apiPost<T>(path: string, body?: unknown, opts?: { withAuth?: boolean; isFormData?: boolean; root?: boolean; headers?: Record<string, string> }): Promise<T> {
  const res = await performRequest('POST', path, { ...opts, body })
  return handle<T>(res)
}

export async function apiPut<T>(path: string, body?: unknown): Promise<T> {
  const res = await performRequest('PUT', path, { body })
  return handle<T>(res)
}

export async function apiPatch<T>(path: string, body?: unknown): Promise<T> {
  const res = await performRequest('PATCH', path, { body })
  return handle<T>(res)
}

export async function apiDelete<T>(path: string): Promise<T> {
  const res = await performRequest('DELETE', path)
  return handle<T>(res)
}

export function buildQuery(params: Record<string, string | number | boolean | null | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== null && value !== undefined && value !== '') {
      search.set(key, String(value))
    }
  }
  const s = search.toString()
  return s ? `?${s}` : ''
}

export { tokenStore }
