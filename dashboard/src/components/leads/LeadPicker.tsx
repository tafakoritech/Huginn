import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { useClickOutside, useDebouncedValue } from '@/hooks'
import { Icon } from '@/components/Icon'
import { StatusBadge } from '@/components/ui/Badge'
import type { Lead } from '@/api/types'

/**
 * Searchable lead autocomplete for the global follow-up form. Searches the
 * backend lead list (`GET /leads?search=...`), which matches company name,
 * city and industry (and the dev mock additionally matches contact name and
 * mobile). Each result row shows enough detail to distinguish same-named
 * leads: company, industry · city · phone and status.
 */
export function LeadPicker({
  value,
  onChange,
  ariaLabel,
}: {
  value: string
  onChange: (id: string) => void
  ariaLabel?: string
}) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Lead | null>(null)
  const debounced = useDebouncedValue(search, 250)

  const query = { page_size: 20, search: debounced.trim() || undefined }
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.leads(query),
    queryFn: () => api.listLeads(query),
    enabled: open,
  })

  const ref = useClickOutside<HTMLDivElement>(() => setOpen(false))

  const pick = (lead: Lead) => {
    setSelected(lead)
    setSearch(lead.company_name)
    setOpen(false)
    onChange(lead.id)
  }

  const clear = () => {
    setSelected(null)
    setSearch('')
    setOpen(false)
    onChange('')
  }

  return (
    <div className="lead-picker" ref={ref}>
      <div className={`lead-picker-input ${open ? 'is-open' : ''}`}>
        <Icon name="search" size={15} />
        <input
          value={selected ? selected.company_name : search}
          onChange={(e) => {
            setSelected(null)
            setSearch(e.target.value)
            setOpen(true)
            if (value) onChange('')
          }}
          onFocus={() => setOpen(true)}
          placeholder={t('followups.searchLead')}
          aria-label={ariaLabel ?? t('followups.lead')}
          autoComplete="off"
        />
        {selected || search ? (
          <button type="button" className="lead-picker-clear" onClick={clear} aria-label={t('common.clear')}>
            <Icon name="x" size={13} />
          </button>
        ) : null}
      </div>

      {open ? (
        <div className="lead-picker-popover" role="listbox" aria-label={t('followups.lead')}>
          {isLoading ? (
            <div className="lead-picker-muted">{t('common.loading')}</div>
          ) : (data?.items ?? []).length === 0 ? (
            <div className="lead-picker-muted">{t('common.noData')}</div>
          ) : (
            (data?.items ?? []).map((l) => (
              <button key={l.id} type="button" role="option" aria-selected={l.id === value} className="lead-picker-row" onClick={() => pick(l)}>
                <span className="lead-picker-name">{l.company_name}</span>
                <span className="lead-picker-meta">{[l.industry, l.city, l.primary_contact?.mobile].filter(Boolean).join(' · ') || '—'}</span>
                <StatusBadge status={l.status} />
              </button>
            ))
          )}
        </div>
      ) : null}
    </div>
  )
}
