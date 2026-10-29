import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

export function useClickOutside<T extends HTMLElement>(onOutside: () => void) {
  const ref = useRef<T>(null)
  useEffect(() => {
    function handle(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onOutside()
      }
    }
    function handleKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onOutside()
    }
    document.addEventListener('mousedown', handle)
    document.addEventListener('keydown', handleKey)
    return () => {
      document.removeEventListener('mousedown', handle)
      document.removeEventListener('keydown', handleKey)
    }
  }, [onOutside])
  return ref
}

export function useDebouncedValue<T>(value: T, delay = 300): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(t)
  }, [value, delay])
  return debounced
}

export function useLocalStorage<T>(key: string, initial: T) {
  const [value, setValue] = useState<T>(() => {
    try {
      const raw = localStorage.getItem(key)
      return raw ? (JSON.parse(raw) as T) : initial
    } catch {
      return initial
    }
  })
  const set = (next: T | ((prev: T) => T)) => {
    setValue((prev) => {
      const v = typeof next === 'function' ? (next as (p: T) => T)(prev) : next
      try {
        localStorage.setItem(key, JSON.stringify(v))
      } catch {
        /* ignore */
      }
      return v
    })
  }
  return [value, set] as const
}

export type UrlState = Record<string, string>

export function useUrlState<T extends UrlState>(defaults: T): [T, (patch: Partial<T>) => void] {
  const [searchParams, setSearchParams] = useSearchParams()
  const state = { ...defaults } as T
  for (const [k, v] of searchParams.entries()) {
    state[k as keyof T] = v as never
  }
  const set = (patch: Partial<T>) => {
    const next = new URLSearchParams(searchParams)
    for (const [k, v] of Object.entries(patch)) {
      if (v === undefined || v === null || v === '') {
        next.delete(k)
      } else {
        next.set(k, String(v))
      }
    }
    const s = next.toString()
    setSearchParams(s ? `?${s}` : '', { replace: true })
  }
  return [state, set]
}
