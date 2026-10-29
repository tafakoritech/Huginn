import '@testing-library/jest-dom/vitest'
import { vi } from 'vitest'

// Force the explicit mock API layer for all component/integration tests.
vi.mock('@/api', async () => {
  const m = await vi.importActual<typeof import('@/mocks/mockApi')>('@/mocks/mockApi')
  return { api: m.mockApi, isMockApi: true }
})

// Component tests use a deterministic auth harness instead of the real
// session-based AuthProvider. PERMISSIONS stays the real one.
vi.mock('@/auth/AuthContext', async () => {
  const perms = await vi.importActual<typeof import('@/auth/permissions')>('@/auth/permissions')
  const harness = await vi.importActual<typeof import('@/test/authHarness')>('@/test/authHarness')
  return { PERMISSIONS: perms.PERMISSIONS, useAuth: harness.useAuth, AuthProvider: harness.AuthProvider }
})

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
window.ResizeObserver = window.ResizeObserver || ResizeObserverStub

// Node 25 exposes a broken experimental localStorage when --localstorage-file
// is missing. Provide a deterministic in-memory implementation instead.
function createStorage(): Storage {
  let data = new Map<string, string>()
  const storage: Storage = {
    get length() {
      return data.size
    },
    clear: () => {
      data = new Map()
    },
    getItem: (k) => (data.has(k) ? data.get(k) ?? null : null),
    key: (i) => Array.from(data.keys())[i] ?? null,
    removeItem: (k) => {
      data.delete(k)
    },
    setItem: (k, v) => {
      data.set(k, String(v))
    },
  }
  return storage
}

const storage = createStorage()
Object.defineProperty(globalThis, 'localStorage', { value: storage, configurable: true })
Object.defineProperty(window, 'localStorage', { value: storage, configurable: true })
