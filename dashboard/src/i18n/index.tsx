import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { pickDictionary, type Dict, type Locale } from './strings'
import { config } from '@/config'

type I18nContextValue = {
  locale: Locale
  dir: 'rtl' | 'ltr'
  t: (key: string, vars?: Record<string, string | number>) => string
  setLocale: (l: Locale) => void
  fa: boolean
}

const I18nContext = createContext<I18nContextValue | null>(null)

function resolvePath(dict: Dict, path: string): string | undefined {
  const parts = path.split('.')
  let cur: unknown = dict
  for (const part of parts) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[part]
  }
  return typeof cur === 'string' ? cur : undefined
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const stored = localStorage.getItem(config.storageKeys.locale) as Locale | null
    return stored && (stored === 'fa' || stored === 'en') ? stored : config.defaultLocale
  })

  const dict = useMemo(() => pickDictionary(locale), [locale])
  const dir: 'rtl' | 'ltr' = locale === 'fa' ? 'rtl' : 'ltr'

  useEffect(() => {
    document.documentElement.setAttribute('lang', locale)
    document.documentElement.setAttribute('dir', dir)
  }, [locale, dir])

  const setLocale = useCallback((l: Locale) => {
    localStorage.setItem(config.storageKeys.locale, l)
    setLocaleState(l)
  }, [])

  const t = useCallback(
    (key: string, vars?: Record<string, string | number>) => {
      let text = resolvePath(dict, key)
      if (text === undefined) {
        const faDict = pickDictionary('fa')
        text = resolvePath(faDict, key)
      }
      if (text === undefined) return key
      if (vars) {
        for (const [k, v] of Object.entries(vars)) {
          text = text.replaceAll(`{${k}}`, String(v))
        }
      }
      return text
    },
    [dict],
  )

  const value = useMemo(
    () => ({ locale, dir, t, setLocale, fa: locale === 'fa' }),
    [locale, dir, t, setLocale],
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}
