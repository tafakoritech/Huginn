import { useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { useI18n } from '@/i18n'
import { useLocalStorage } from '@/hooks'

export function AppShell() {
  const { t, fa } = useI18n()
  const [collapsed, setCollapsed] = useLocalStorage('huginn.sidebar', false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const { pathname } = useLocation()

  const pageTitle = pageTitleFor(pathname, t)
  const currentPage = pageTitle ?? t('nav.dashboard')

  return (
    <div className="app-shell">
      <Sidebar
        collapsed={collapsed}
        onToggle={() => setCollapsed((c) => !c)}
        mobileOpen={mobileOpen}
        onMobileClose={() => setMobileOpen(false)}
      />
      <div className="app-main" dir={fa ? 'rtl' : 'ltr'}>
        <TopBar onMobileMenu={() => setMobileOpen(true)} currentPage={currentPage} />
        <main className="app-content" id="main-content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

function pageTitleFor(pathname: string, t: (k: string) => string): string | null {
  const segments = pathname.split('/').filter(Boolean)
  const first = segments[0] ?? ''
  switch (first) {
    case 'dashboard':
      return t('nav.dashboard')
    case 'imports':
      return t('nav.inputCategories')
    case 'documents':
      return t('nav.documents')
    case 'leads':
      if (segments[1] === 'qualified') return t('nav.qualifiedLeads')
      if (segments[1]) return t('leads.title')
      return t('nav.allLeads')
    case 'duplicates':
      return t('nav.duplicates')
    case 'assignments':
      return t('nav.assignments')
    case 'followups':
      return t('nav.followups')
    case 'reports':
      return t('nav.contactReports')
    case 'agents':
      return t('nav.agents')
    case 'audit':
      return t('nav.audit')
    case 'change-history':
      return t('nav.changeHistory')
    case 'settings':
      if (segments[1]) return t(`nav.${segments[1]}`)
      return t('nav.settings')
    default:
      return null
  }
}
