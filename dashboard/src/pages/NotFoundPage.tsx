import { Link } from 'react-router-dom'
import { useI18n } from '@/i18n'
import { Icon } from '@/components/Icon'

export function NotFoundPage() {
  const { t } = useI18n()
  return (
    <div className="state-block" style={{ minHeight: '60vh' }}>
      <Icon name="alertTriangle" size={48} />
      <h3>404</h3>
      <p>{t('errors.notFound')}</p>
      <Link to="/dashboard" className="btn btn-primary">
        {t('errors.goHome')}
      </Link>
    </div>
  )
}
