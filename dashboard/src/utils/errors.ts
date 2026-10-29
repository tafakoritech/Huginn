import { ApiError, NetworkError } from '@/api/client'
import type { Dict } from '@/i18n/strings'

const codeToKey: Record<string, string> = {
  UNAUTHORIZED: 'errors.unauthorized',
  FORBIDDEN: 'errors.forbidden',
  NOT_FOUND: 'errors.notFound',
  VALIDATION_ERROR: 'errors.validation',
  BAD_REQUEST: 'errors.validation',
  CONFLICT: 'errors.conflict',
  RATE_LIMITED: 'errors.rateLimited',
  MANDATORY_REPORT_PENDING: 'errors.mandatoryReportPending',
  INVALID_STATE_TRANSITION: 'errors.invalidTransition',
  DUPLICATE_EXPORT: 'errors.duplicateExport',
  PAYLOAD_TOO_LARGE: 'errors.payloadTooLarge',
  UNSUPPORTED_MEDIA_TYPE: 'errors.unsupportedMedia',
  INTERNAL_ERROR: 'errors.generic',
}

export function errorMessage(err: unknown, t: (key: string) => string): string {
  if (err instanceof ApiError) {
    const key = codeToKey[err.code]
    if (key) return t(key)
    return err.message || t('errors.generic')
  }
  if (err instanceof NetworkError) return t('errors.network')
  if (err instanceof Error) {
    if ((err as Error & { code?: string }).code) {
      const key = codeToKey[(err as Error & { code: string }).code]
      if (key) return t(key)
    }
    return err.message || t('errors.generic')
  }
  return t('errors.generic')
}

export function isErrorCode(err: unknown, code: string): boolean {
  if (err instanceof ApiError) return err.code === code
  if (err instanceof Error) return (err as Error & { code?: string }).code === code
  return false
}

export type LocalizedFn = (key: string) => string
export type _Dict = Dict
