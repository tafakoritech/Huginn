import { useState } from 'react'
import { useI18n } from '@/i18n'
import { Icon } from '@/components/Icon'
import { JalaliDatePicker } from '@/components/ui/JalaliDatePicker'
import { Textarea } from '@/components/ui/Field'
import { Button } from '@/components/ui/Button'
import { LeadPicker } from '@/components/leads/LeadPicker'
import { addDaysJalali, jalaliToIsoDate, jalaliToday, type JalaliDate } from '@/utils/jalali'

export type FollowUpFormValue = {
  lead_id?: string
  type: string
  due_at: string
  description?: string
}

const FOLLOWUP_TYPE_OPTIONS = [
  { value: 'CALL', icon: 'phone' as const },
  { value: 'FOLLOW_UP', icon: 'clock' as const },
  { value: 'MESSAGE', icon: 'message' as const },
  { value: 'EMAIL', icon: 'mail' as const },
] as const

/**
 * Reusable follow-up creation form. Two submit modes backed by two backend
 * contracts:
 *  - `lead`: the lead is fixed (shown at the top); submit value carries no
 *    `lead_id` because the nested endpoint already reads it from the URL.
 *  - `global`: the user picks a lead via the searchable [LeadPicker]; submit
 *    value always carries `lead_id` for the global POST /followups endpoint.
 */
export function FollowUpForm({
  mode,
  leadName,
  onCancel,
  onSubmit,
  loading,
}: {
  mode: 'lead' | 'global'
  leadName?: string
  onCancel: () => void
  onSubmit: (value: FollowUpFormValue) => void
  loading: boolean
}) {
  const { t } = useI18n()
  const [leadId, setLeadId] = useState('')
  const [dueAt, setDueAt] = useState<JalaliDate | null>(() => addDaysJalali(jalaliToday(), 1))
  const [type, setType] = useState<(typeof FOLLOWUP_TYPE_OPTIONS)[number]['value']>('CALL')
  const [description, setDescription] = useState('')

  const leadMissing = mode === 'global' && !leadId

  const submit = () => {
    if (!dueAt) return
    if (leadMissing) return
    onSubmit({
      ...(mode === 'global' ? { lead_id: leadId } : {}),
      type,
      due_at: jalaliToIsoDate(dueAt.year, dueAt.month, dueAt.day),
      description: description.trim() || undefined,
    })
  }

  return (
    <div>
      {mode === 'lead' && leadName ? (
        <div className="followup-company">
          <Icon name="package" size={16} />
          <span>{leadName}</span>
        </div>
      ) : null}

      {mode === 'global' ? (
        <div className="field">
          <label className="field-label">{t('followups.lead')}</label>
          <LeadPicker value={leadId} onChange={setLeadId} />
        </div>
      ) : null}

      <div className="field">
        <label className="field-label">{t('followups.type')}</label>
        <div className="followup-types" role="radiogroup" aria-label={t('followups.type')}>
          {FOLLOWUP_TYPE_OPTIONS.map((o) => {
            const selected = type === o.value
            return (
              <button
                key={o.value}
                type="button"
                role="radio"
                aria-checked={selected}
                className={`followup-type-btn ${selected ? 'is-selected' : ''}`}
                onClick={() => setType(o.value)}
              >
                <Icon name={o.icon} size={16} />
                {t(`followups.types.${o.value}`)}
              </button>
            )
          })}
        </div>
      </div>

      <div className="field">
        <label className="field-label">{t('followups.dueDate')}</label>
        <JalaliDatePicker value={dueAt} onChange={setDueAt} />
      </div>

      <div className="field">
        <label className="field-label">{t('followups.description')}</label>
        <Textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder={t('leads.nextAction')} />
      </div>

      <div className="form-actions">
        <Button onClick={onCancel}>{t('common.cancel')}</Button>
        <Button variant="primary" loading={loading} disabled={!dueAt || leadMissing} onClick={submit}>
          {t('common.save')}
        </Button>
      </div>
    </div>
  )
}
