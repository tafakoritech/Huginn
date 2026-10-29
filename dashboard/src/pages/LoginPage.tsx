import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useAuth } from '@/auth/AuthContext'
import { useI18n } from '@/i18n'
import { config } from '@/config'
import { errorMessage } from '@/utils/errors'
import { Icon } from '@/components/Icon'

const schema = z.object({
  login: z.string().min(1),
  password: z.string().min(1),
})

type FormValues = z.infer<typeof schema>

export function LoginPage() {
  const { t, fa } = useI18n()
  const { login } = useAuth()
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { login: '', password: '' },
  })

  const onSubmit = async (values: FormValues) => {
    setError(null)
    setSubmitting(true)
    try {
      await login(values.login, values.password)
      navigate('/dashboard')
    } catch (err) {
      setError(errorMessage(err, t))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="login-page" dir={fa ? 'rtl' : 'ltr'}>
      <div className="login-card">
        <div className="login-brand">
          <img src="/raven.png" alt={config.appName} className="brand-logo" />
          <h1>{t('auth.welcome')}</h1>
          <p>{t('auth.loginSubtitle')}</p>
        </div>

        {error ? (
          <div className="login-error" role="alert">
            {error}
          </div>
        ) : null}

        <form onSubmit={handleSubmit(onSubmit)} noValidate>
          <div className="field">
            <label className="field-label" htmlFor="login">
              {t('auth.username')}
            </label>
            <input
              id="login"
              className={`input ${errors.login ? 'has-error' : ''}`.trim()}
              autoComplete="username"
              dir="ltr"
              style={{ textAlign: fa ? 'right' : 'left' }}
              {...register('login')}
            />
            {errors.login ? (
              <span className="field-error" role="alert">
                {t('common.required')}
              </span>
            ) : null}
          </div>

          <div className="field">
            <label className="field-label" htmlFor="password">
              {t('auth.password')}
            </label>
            <input
              id="password"
              type="password"
              className={`input ${errors.password ? 'has-error' : ''}`.trim()}
              autoComplete="current-password"
              dir="ltr"
              style={{ textAlign: fa ? 'right' : 'left' }}
              {...register('password')}
            />
            {errors.password ? (
              <span className="field-error" role="alert">
                {t('common.required')}
              </span>
            ) : null}
          </div>

          <button type="submit" className="btn btn-primary btn-lg btn-block" disabled={submitting}>
            {submitting ? <span className="spinner" /> : <Icon name="lock" size={16} />}
            {t('auth.loginBtn')}
          </button>
        </form>
      </div>
    </div>
  )
}
