import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api'
import { useI18n } from '@/i18n'
import { errorMessage } from '@/utils/errors'
import { useToastApi } from '@/components/ui/Toast'
import { Dialog } from '@/components/ui/Dialog'
import { FollowUpForm, type FollowUpFormValue } from './FollowUpForm'

/**
 * Shared "Create follow-up" dialog used from two contexts:
 *  - `lead`: the lead is pre-selected (Lead Detail); posts to
 *    `POST /leads/{leadID}/followups` without a `lead_id` in the body.
 *  - `global`: the user picks a lead (Follow-ups menu); posts to
 *    `POST /followups` with `lead_id` in the body.
 *
 * Owns the create mutation, the success toast and the query invalidation so
 * every entry point refreshes the same follow-up lists after a successful save.
 */
export function CreateFollowUpDialog({
  open,
  onClose,
  mode,
  leadId,
  leadName,
}: {
  open: boolean
  onClose: () => void
  mode: 'lead' | 'global'
  leadId?: string
  leadName?: string
}) {
  const { t } = useI18n()
  const toast = useToastApi()
  const qc = useQueryClient()

  const create = useMutation({
    mutationFn: (value: FollowUpFormValue) => {
      if (mode === 'lead') {
        if (!leadId) throw new Error('lead id is required in lead mode')
        return api.addFollowUp(leadId, value)
      }
      if (!value.lead_id) throw new Error('lead_id is required in global mode')
      return api.createFollowUp({ ...value, lead_id: value.lead_id })
    },
    onSuccess: () => {
      toast.success(t('followups.created'))
      onClose()
      void qc.invalidateQueries({ queryKey: ['followups'] })
      if (mode === 'lead' && leadId) {
        void qc.invalidateQueries({ queryKey: ['lead', leadId] })
        void qc.invalidateQueries({ queryKey: ['leads'] })
      }
    },
    onError: (e) => toast.error(errorMessage(e, t)),
  })

  return (
    <Dialog open={open} onClose={onClose} title={t('followups.create')}>
      <FollowUpForm
        mode={mode}
        leadName={mode === 'lead' ? leadName : undefined}
        onCancel={onClose}
        onSubmit={(value) => create.mutate(value)}
        loading={create.isPending}
      />
    </Dialog>
  )
}
