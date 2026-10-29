import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { Icon } from '@/components/Icon'
import { useDebouncedValue, useClickOutside } from '@/hooks'
import { toFaDigits } from '@/utils/format'
import { StatusBadge } from '@/components/ui/Badge'

export function GlobalSearch() {
  const { t } = useI18n()
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const debounced = useDebouncedValue(query, 300)
  const navigate = useNavigate()
  const inputRef = useRef<HTMLInputElement>(null)

  const { data, isFetching } = useQuery({
    queryKey: queryKeys.search(debounced),
    queryFn: () => api.globalSearch(debounced),
    enabled: debounced.length >= 2,
    staleTime: 30_000,
  })

  const ref = useClickOutside<HTMLDivElement>(() => setOpen(false))

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        inputRef.current?.focus()
        setOpen(true)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  const totalResults = (data?.leads.length ?? 0) + (data?.users.length ?? 0)

  return (
    <div className="topbar-search" ref={ref}>
      <div className="dropdown" style={{ width: '100%' }}>
        <div className="search-input-wrap">
          <Icon name="search" />
          <input
            ref={inputRef}
            type="search"
            className="input"
            placeholder={t('search.placeholder')}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setOpen(true)
            }}
            onFocus={() => setOpen(true)}
            aria-label={t('header.globalSearch')}
            style={{ paddingInlineEnd: 40 }}
          />
          <span className="kbd-hint" style={{ position: 'absolute', insetInlineEnd: 8, top: '50%', transform: 'translateY(-50%)' }}>
            Ctrl+K
          </span>
        </div>
        {open && (debounced.length >= 2 || query.length > 0) ? (
          <div className="menu menu-start search-panel" style={{ position: 'absolute', top: 'calc(100% + 6px)', insetInlineStart: 0, insetInlineEnd: 'auto' }}>
            {isFetching ? (
              <div className="loading-block" style={{ padding: 16 }}>
                <span className="spinner" /> {t('common.loading')}
              </div>
            ) : totalResults === 0 ? (
              <div className="state-block" style={{ padding: 20 }}>
                <p className="muted">{t('search.noResults')}</p>
              </div>
            ) : (
              <>
                {data?.leads.length ? (
                  <div>
                    <div className="search-group-label">{t('search.leads')}</div>
                    {data.leads.map((l) => (
                      <button
                        key={l.id}
                        type="button"
                        className="search-result"
                        onClick={() => {
                          setOpen(false)
                          navigate(`/leads/${l.id}`)
                        }}
                      >
                        <span className="flex-1 flex items-center justify-between gap-3">
                          <span className="flex flex-col">
                            <span className="sr-title">{l.company_name}</span>
                            <span className="sr-sub">
                              {l.city ?? ''} · {t(`status.${l.status}`)}
                            </span>
                          </span>
                          <StatusBadge status={l.status} />
                        </span>
                      </button>
                    ))}
                  </div>
                ) : null}
                {data?.users.length ? (
                  <div>
                    <div className="search-group-label">{t('search.users')}</div>
                    {data.users.map((u) => (
                      <button
                        key={u.id}
                        type="button"
                        className="search-result"
                        onClick={() => {
                          setOpen(false)
                          navigate(`/settings/users?id=${u.id}`)
                        }}
                      >
                        <Icon name="user" size={15} />
                        <span className="flex flex-col">
                          <span className="sr-title">{u.name}</span>
                          <span className="sr-sub">{u.username}</span>
                        </span>
                      </button>
                    ))}
                  </div>
                ) : null}
              </>
            )}
          </div>
        ) : null}
      </div>
    </div>
  )
}

export function toFa(value: number, locale: string) {
  return locale === 'fa' ? toFaDigits(value) : String(value)
}
