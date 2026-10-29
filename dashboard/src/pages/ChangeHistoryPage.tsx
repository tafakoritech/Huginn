import { Fragment, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { formatDateTime } from '@/utils/format'
import { PageHeader } from '@/components/ui/misc'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import type { ChangeHistoryItem } from '@/api/types'

export function ChangeHistoryPage() {
  const { t, locale } = useI18n()
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.changeHistory(1),
    queryFn: () => api.recentLeadChanges(100, 0),
  })

  const items = data?.items ?? []

  return (
    <>
      <PageHeader title={t('versions.adminTitle')} subtitle={t('versions.adminSubtitle')} />

      <div className="card">
        <div className="table-scroll">
          <table className="table">
            <thead>
              <tr>
                <th>{t('versions.lead')}</th>
                <th>{t('versions.version')}</th>
                <th>{t('versions.actor')}</th>
                <th>{t('versions.time')}</th>
                <th>{t('versions.note')}</th>
                <th>{t('versions.field')}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="table-empty">
                    <p className="muted">{t('common.loading')}</p>
                  </td>
                </tr>
              ) : items.length === 0 ? (
                <tr>
                  <td colSpan={7} className="table-empty">
                    <p className="muted">{t('common.noData')}</p>
                  </td>
                </tr>
              ) : (
                items.map((item) => {
                  const isOpen = !!expanded[item.id]
                  const firstChange = (item.changes ?? [])[0]
                  return (
                    <Fragment key={item.id}>
                      <tr>
                        <td className="strong">
                          <Link to={`/leads/${item.lead_id}`} className="link">
                            {item.lead_name}
                          </Link>
                        </td>
                        <td>
                          <Badge tone="brand">{t('versions.versionLabel', { n: String(item.version) })}</Badge>
                        </td>
                        <td>
                          {item.actor?.name ?? t('common.system')}
                          {item.changed_by_role ? (
                            <span className="small secondary block">{item.changed_by_role}</span>
                          ) : null}
                        </td>
                        <td>{formatDateTime(item.created_at, locale)}</td>
                        <td>{item.note ?? <span className="muted">—</span>}</td>
                        <td className="small">{firstChange ? firstChange.field : '—'}</td>
                        <td>
                          <Button size="sm" variant="ghost" icon="chevronDown" onClick={() => setExpanded((e) => ({ ...e, [item.id]: !isOpen }))} aria-expanded={isOpen}>
                            {t('versions.details')}
                          </Button>
                        </td>
                      </tr>
                      {isOpen ? (
                        <tr>
                          <td colSpan={7}>
                            <VersionChanges item={item} />
                          </td>
                        </tr>
                      ) : null}
                    </Fragment>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
    </>
  )
}

function VersionChanges({ item }: { item: ChangeHistoryItem }) {
  const { t } = useI18n()
  const changes = item.changes ?? []
  return (
    <div className="version-diff">
      {changes.length === 0 ? (
        <p className="muted small">{t('versions.noChanges')}</p>
      ) : (
        changes.map((ch, i) => (
          <div className="version-diff-row" key={i}>
            <span className="mono small diff-field" dir="ltr">{ch.field}</span>
            <span className="diff-old small">{typeof ch.old === 'object' ? JSON.stringify(ch.old) : String(ch.old ?? '—')}</span>
            <span className="diff-arrow">←</span>
            <span className="diff-new small">{typeof ch.new === 'object' ? JSON.stringify(ch.new) : String(ch.new ?? '—')}</span>
          </div>
        ))
      )}
    </div>
  )
}
