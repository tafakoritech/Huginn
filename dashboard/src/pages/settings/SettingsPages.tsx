import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api'
import { queryKeys } from '@/api/keys'
import { useI18n } from '@/i18n'
import { toFaDigits } from '@/utils/format'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { PageHeader } from '@/components/ui/misc'
import { LoadingState } from '@/components/ui/States'
import { Button } from '@/components/ui/Button'
import { Checkbox } from '@/components/ui/Field'
import { Icon } from '@/components/Icon'
import { Badge } from '@/components/ui/Badge'

const DIM_LABELS = ['business_fit', 'contactability', 'scale', 'engagement', 'authority', 'need', 'timing']

export function ScoringSettingsPage() {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: queryKeys.scoringSettings, queryFn: () => api.getScoringSettings() })
  const [weights, setWeights] = useState<Record<string, number>>({})
  const [dirty, setDirty] = useState(false)

  const save = useMutation({
    mutationFn: () => api.updateScoringSettings(weights),
    onSuccess: () => {
      toast.success(t('settings.weightUpdated'))
      setDirty(false)
      void qc.invalidateQueries({ queryKey: queryKeys.scoringSettings })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  if (isLoading || !data) return <LoadingState />

  const current = weights
  const total = DIM_LABELS.reduce((a, key) => a + (current[key] ?? data.dimensions.find((d) => d.key === key)?.weight ?? 0), 0)
  const valid = total === 100

  return (
    <>
      <PageHeader
        title={t('settings.scoringTitle')}
        subtitle={t('settings.scoringSubtitle')}
        breadcrumbs={[{ label: t('nav.settings'), to: '/settings/scoring' }]}
        actions={
          dirty ? (
            <Button variant="primary" disabled={!valid} loading={save.isPending} onClick={() => save.mutate()}>
              {t('common.save')}
            </Button>
          ) : null
        }
      />

      <div className="settings-section">
        <div className="card">
          <div className="card-header">
            <h3>{t('settings.scoringTitle')}</h3>
            <Badge tone={valid ? 'success' : 'error'}>
              {t('settings.totalWeight')}: {toFaDigits(total)}/100
            </Badge>
          </div>
          <div className="card-body">
            {DIM_LABELS.map((key) => {
              const dim = data.dimensions.find((d) => d.key === key)
              const weight = current[key] ?? dim?.weight ?? 0
              return (
                <div className="score-dim" key={key}>
                  <span className="dim-name">{t(`leads.scoreDimensions.${key}`)}</span>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <input
                      type="number"
                      className="input"
                      min={0}
                      max={100}
                      value={weight}
                      dir="ltr"
                      style={{ width: 70, height: 30, textAlign: 'center' }}
                      onChange={(e) => {
                        setWeights((w) => ({ ...w, [key]: Number(e.target.value) }))
                        setDirty(true)
                      }}
                      aria-label={t(`leads.scoreDimensions.${key}`)}
                    />
                  </div>
                  <span className="dim-score">%{toFaDigits(weight)}</span>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </>
  )
}

export function QualificationSettingsPage() {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: queryKeys.qualificationSettings, queryFn: () => api.getQualificationSettings() })
  const [draft, setDraft] = useState<Record<string, unknown> | null>(null)

  const save = useMutation({
    mutationFn: () => api.updateQualificationSettings(draft ?? {}),
    onSuccess: () => {
      toast.success(t('common.save'))
      setDraft(null)
      void qc.invalidateQueries({ queryKey: queryKeys.qualificationSettings })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  if (isLoading || !data) return <LoadingState />

  const current = draft ?? (data as unknown as Record<string, unknown>)

  const toggles = [
    { key: 'business_relevance_required' as const, labelKey: 'settings.businessRelevance' },
    { key: 'meaningful_contact_required' as const, labelKey: 'settings.meaningfulContact' },
    { key: 'decision_maker_required' as const, labelKey: 'settings.decisionMakerReq' },
    { key: 'need_interest_required' as const, labelKey: 'settings.needInterestReq' },
    { key: 'next_action_required' as const, labelKey: 'settings.nextActionReq' },
  ]

  const set = (key: string, value: unknown) => setDraft((d) => ({ ...(d ?? (data as unknown as Record<string, unknown>)), [key]: value }))

  return (
    <>
      <PageHeader
        title={t('settings.qualTitle')}
        subtitle={t('settings.qualSubtitle')}
        breadcrumbs={[{ label: t('nav.settings'), to: '/settings/qualification' }]}
        actions={
          <Button variant="primary" loading={save.isPending} onClick={() => save.mutate()}>
            {t('common.save')}
          </Button>
        }
      />

      <div className="settings-section">
        <div className="card">
          <div className="card-header">
            <h3>{t('settings.qualTitle')}</h3>
          </div>
          <div className="card-body">
            <div className="field" style={{ maxWidth: 220 }}>
              <label className="field-label">{t('settings.minimumScore')}</label>
              <input type="number" className="input" dir="ltr" value={String(current.minimum_score ?? 70)} onChange={(e) => set('minimum_score', Number(e.target.value))} />
            </div>
            {toggles.map((tg) => (
              <div className="mb-3" key={tg.key}>
                <Checkbox label={t(tg.labelKey)} checked={!!current[tg.key]} onChange={(v) => set(tg.key, v)} />
              </div>
            ))}
            <p className="small muted mt-4">
              <Icon name="info" size={13} />
              {t('settings.effectExplanation')}
            </p>
          </div>
        </div>
      </div>
    </>
  )
}

export function LLMSettingsPage() {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: queryKeys.llmSettings, queryFn: () => api.getLlmSettings() })
  const [baseUrl, setBaseUrl] = useState('')
  const [timeout, setTimeoutVal] = useState<number>(60)
  const [testing, setTesting] = useState(false)

  const save = useMutation({
    mutationFn: () =>
      api.updateLlmSettings({
        base_url: baseUrl || undefined,
        timeout_seconds: timeout,
      }),
    onSuccess: () => {
      toast.success(t('common.save'))
      void qc.invalidateQueries({ queryKey: queryKeys.llmSettings })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  const test = async () => {
    setTesting(true)
    try {
      const res = await api.testLlmConnection()
      if (res.ok) toast.success(t('settings.connectionOk'))
      else toast.error(t('settings.connectionFailed'))
    } catch (e) {
      toast.error(errorMessage(e, t))
    } finally {
      setTesting(false)
    }
  }

  if (isLoading || !data) return <LoadingState />

  return (
    <>
      <PageHeader
        title={t('settings.llmTitle')}
        subtitle={t('settings.llmSubtitle')}
        breadcrumbs={[{ label: t('nav.settings'), to: '/settings/llm' }]}
      />
      <div className="settings-section">
        <div className="card">
          <div className="card-header">
            <h3>{t('settings.llmTitle')}</h3>
            <Badge tone={data.structured_output_enabled ? 'success' : 'neutral'}>{t('settings.structuredOutput')}</Badge>
          </div>
          <div className="card-body">
            <div className="meta-grid mb-5">
              <div className="meta-item">
                <div className="meta-label">{t('settings.provider')}</div>
                <div className="meta-value">{data.provider}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">{t('settings.model')}</div>
                <div className="meta-value mono" dir="ltr">{data.model}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">{t('settings.promptVersion')}</div>
                <div className="meta-value mono" dir="ltr">{data.prompt_version}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">{t('settings.timeout')}</div>
                <div className="meta-value">{toFaDigits(data.timeout_seconds)}</div>
              </div>
            </div>

            <div className="form-row mb-4">
              <div className="field">
                <label className="field-label">{t('settings.baseUrl')}</label>
                <input className="input" dir="ltr" defaultValue={data.base_url} onChange={(e) => setBaseUrl(e.target.value)} />
              </div>
              <div className="field">
                <label className="field-label">{t('settings.timeout')}</label>
                <input type="number" className="input" dir="ltr" defaultValue={data.timeout_seconds} onChange={(e) => setTimeoutVal(Number(e.target.value))} />
              </div>
            </div>

            <p className="small muted mb-4">
              <Icon name="lock" size={13} />
              {t('settings.apiKeySecret')}
            </p>

            <div className="flex gap-2">
              <Button variant="primary" loading={save.isPending} onClick={() => save.mutate()}>
                {t('common.save')}
              </Button>
              <Button loading={testing} onClick={() => void test()}>
                {t('settings.testConnection')}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

export function MuninnSettingsPage() {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: queryKeys.muninnSettings, queryFn: () => api.getMuninnSettings() })
  const [baseUrl, setBaseUrl] = useState('')

  const save = useMutation({
    mutationFn: () => api.updateMuninnSettings({ base_url: baseUrl || undefined }),
    onSuccess: () => {
      toast.success(t('common.save'))
      void qc.invalidateQueries({ queryKey: queryKeys.muninnSettings })
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  if (isLoading || !data) return <LoadingState />

  return (
    <>
      <PageHeader title={t('settings.muninnTitle')} subtitle={t('settings.muninnSubtitle')} breadcrumbs={[{ label: t('nav.settings'), to: '/settings/muninn' }]} />
      <div className="settings-section">
        <div className="card">
          <div className="card-header">
            <h3>{t('settings.muninnTitle')}</h3>
            <Badge tone={data.enabled ? 'success' : 'neutral'}>{data.enabled ? t('common.active') : t('common.inactive')}</Badge>
          </div>
          <div className="card-body">
            <div className="meta-grid mb-5">
              <div className="meta-item">
                <div className="meta-label">{t('settings.baseUrl')}</div>
                <div className="meta-value mono" dir="ltr" style={{ fontSize: 12 }}>{data.base_url}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">{t('settings.exportEndpoint')}</div>
                <div className="meta-value mono" dir="ltr" style={{ fontSize: 12 }}>{data.export_endpoint}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">{t('settings.timeout')}</div>
                <div className="meta-value">{toFaDigits(data.timeout_seconds)}</div>
              </div>
            </div>
            <div className="field mb-4">
              <label className="field-label">{t('settings.baseUrl')}</label>
              <input className="input" dir="ltr" defaultValue={data.base_url} onChange={(e) => setBaseUrl(e.target.value)} />
            </div>
            <Button variant="primary" loading={save.isPending} onClick={() => save.mutate()}>
              {t('common.save')}
            </Button>
          </div>
        </div>
      </div>
    </>
  )
}

export function SystemSettingsPage() {
  const { t } = useI18n()
  const { data, isLoading } = useQuery({ queryKey: queryKeys.systemConfig, queryFn: () => api.getSystemConfig() })

  if (isLoading || !data) return <LoadingState />

  return (
    <>
      <PageHeader title={t('settings.systemTitle')} subtitle={t('settings.systemSubtitle')} breadcrumbs={[{ label: t('nav.settings'), to: '/settings/system' }]} />
      <div className="settings-section">
        <div className="card">
          <div className="card-header">
            <h3>{t('settings.systemTitle')}</h3>
          </div>
          <div className="card-body">
            <div className="meta-grid">
              <div className="meta-item">
                <div className="meta-label">{t('app.name')}</div>
                <div className="meta-value">{data.app_name}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">Version</div>
                <div className="meta-value mono" dir="ltr">{data.version}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">Timezone</div>
                <div className="meta-value" dir="ltr">{data.timezone}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">Storage</div>
                <div className="meta-value">{data.storage_engine}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">{t('imports.processing')}</div>
                <div className="meta-value">{data.queue}</div>
              </div>
              <div className="meta-item">
                <div className="meta-label">Workers</div>
                <div className="meta-value">{toFaDigits(data.worker_count)}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
