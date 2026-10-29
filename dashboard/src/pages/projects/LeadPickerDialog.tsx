import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits } from '@/utils/format'
import { Dialog } from '@/components/ui/Dialog'
import { Button } from '@/components/ui/Button'
import { Input, Select } from '@/components/ui/Field'
import { StatusBadge } from '@/components/ui/Badge'
import { Pagination } from '@/components/ui/Pagination'
import type { PaginationMeta } from '@/api/types'

export function LeadPickerDialog({
  open,
  onClose,
  attachedLeadIds,
  onAttach,
}: {
  open: boolean
  onClose: () => void
  attachedLeadIds: string[]
  onAttach: (leadIds: string[]) => Promise<void>
}) {
  const { t } = useI18n()
  const [search, setSearch] = useState('')
  const [industry, setIndustry] = useState('')
  const [city, setCity] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<Record<string, boolean>>({})
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) return
    setSearch('')
    setIndustry('')
    setCity('')
    setStatus('')
    setPage(1)
    setSelected({})
  }, [open])

  const query = useMemo(
    () => ({
      page,
      page_size: 10,
      search: search || undefined,
      industry: industry || undefined,
      city: city || undefined,
      status: status || undefined,
    }),
    [page, search, industry, city, status],
  )

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.leads(query),
    queryFn: () => api.listLeads(query),
    enabled: open,
  })

  const selectedCount = Object.keys(selected).filter((id) => selected[id]).length

  const toggle = (id: string, checked: boolean) => {
    setSelected((prev) => {
      const next = { ...prev }
      if (checked) next[id] = true
      else delete next[id]
      return next
    })
  }

  const submit = async () => {
    const ids = Object.keys(selected).filter((id) => selected[id])
    if (ids.length === 0) return
    setSubmitting(true)
    try {
      await onAttach(ids)
      setSelected({})
      setPage(1)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="lg"
      title={t('projects.selectLeadsTitle')}
      footer={
        <>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button variant="primary" loading={submitting} disabled={selectedCount === 0} onClick={() => void submit()}>
            {t('projects.attachSelected')} {selectedCount > 0 ? `(${toFaDigits(selectedCount)})` : ''}
          </Button>
        </>
      }
    >
      <div className="form-row mb-4">
        <Input label={t('projects.searchCompany')} value={search} onChange={(e) => { setSearch(e.target.value); setPage(1) }} />
        <Input label={t('leads.industry')} value={industry} onChange={(e) => { setIndustry(e.target.value); setPage(1) }} />
      </div>
      <div className="form-row mb-4">
        <Input label={t('leads.city')} value={city} onChange={(e) => { setCity(e.target.value); setPage(1) }} />
        <Select label={t('leads.status')} value={status} onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
          <option value="">{t('common.all')}</option>
          {['NEW', 'READY', 'ASSIGNED', 'CONTACTING', 'FOLLOW_UP', 'QUALIFIED', 'DISQUALIFIED', 'EXPORTED_TO_MUNINN'].map((s) => (
            <option key={s} value={s}>
              {t(`status.${s}`)}
            </option>
          ))}
        </Select>
      </div>

      <div className="picker-list" style={{ minHeight: 240 }}>
        {isLoading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 5 }, (_, i) => (
              <div className="skeleton skeleton-row" key={i} style={{ height: 38 }} />
            ))}
          </div>
        ) : (data?.items ?? []).length === 0 ? (
          <p className="muted small">{t('projects.noLeadsFound')}</p>
        ) : (
          (data?.items ?? []).map((l) => {
            const attached = attachedLeadIds.includes(l.id)
            const checked = !!selected[l.id] || attached
            return (
              <label key={l.id} className={`picker-row ${attached ? 'picker-row-muted' : ''}`}>
                <input type="checkbox" disabled={attached} checked={checked} onChange={(e) => toggle(l.id, e.target.checked)} />
                <span className="picker-row-main">
                  {attached ? (
                    <span className="strong">{l.company_name}</span>
                  ) : (
                    <Link to={`/leads/${l.id}`} className="strong" style={{ color: 'var(--text-1)' }} onClick={(e) => e.preventDefault()}>
                      {l.company_name}
                    </Link>
                  )}
                  <span className="small muted">{[l.industry, l.city].filter(Boolean).join(' · ') || '—'}</span>
                </span>
                <StatusBadge status={l.status} />
                <span className="small secondary">{l.owner?.name ?? l.owner_name ?? '—'}</span>
              </label>
            )
          })
        )}
      </div>

      {data?.meta ? <Pagination meta={data.meta as PaginationMeta} onPage={setPage} /> : null}
    </Dialog>
  )
}
