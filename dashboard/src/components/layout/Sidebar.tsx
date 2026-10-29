import { Link, useLocation } from 'react-router-dom'
import { Icon, type IconName } from '@/components/Icon'
import { useI18n } from '@/i18n'
import { useAuth } from '@/auth/AuthContext'

type NavLeaf = { key: string; label: string; to: string; icon: IconName }
type NavGroup = { label: string; children: NavLeaf[] }
type NavEntry = NavLeaf | NavGroup

function isGroup(item: NavEntry): item is NavGroup {
  return (item as NavGroup).children !== undefined
}

// isNavActive matches the CURRENT full location (pathname + search) against a
// nav target. Links with a query (e.g. `/leads?status=NEW`) only activate on an
// exact search match. Plain links activate on the exact path or a child route,
// but a plain link is suppressed when one of the dedicated query links matches
// the current search (so `/leads?status=NEW` only highlights "جدید").
function isNavActive(to: string, pathname: string, search: string, queryTargets: string[]): boolean {
  const qIdx = to.indexOf('?')
  const path = qIdx === -1 ? to : to.slice(0, qIdx)
  const query = qIdx === -1 ? '' : to.slice(qIdx + 1)

  if (query !== '') {
    return pathname === path && search === `?${query}`
  }
  if (pathname === path) {
    return search === '' || !queryTargets.includes(search)
  }
  // Collection pages stay highlighted on their child routes, except when the
  // child has its own dedicated item (e.g. /leads/qualified).
  if (path === '/leads' && pathname.startsWith('/leads/qualified')) return false
  if (pathname.startsWith(path + '/')) return true
  return false
}

function useNavItems(): NavEntry[] {
  const { t } = useI18n()
  const { can } = useAuth()
  const items: NavEntry[] = [
    { label: t('nav.inputs'), children: [
      { key: 'imports', label: t('nav.inputCategories'), to: '/imports', icon: 'import' },
      { key: 'documents', label: t('nav.documents'), to: '/documents', icon: 'documents' },
    ]},
    {
      label: t('nav.leads'),
      children: [
        { key: 'leads', label: t('nav.allLeads'), to: '/leads', icon: 'leads' },
        { key: 'leads-new', label: t('nav.newLeads'), to: '/leads?status=NEW', icon: 'plus' },
        { key: 'leads-unassigned', label: t('nav.unassignedLeads'), to: '/leads?unassigned=1', icon: 'users' },
        { key: 'leads-review', label: t('nav.reviewRequired'), to: '/leads?review=1', icon: 'alert' },
        { key: 'qualified', label: t('nav.qualifiedLeads'), to: '/leads/qualified', icon: 'checkCircle' },
        { key: 'disqualified', label: t('nav.disqualifiedLeads'), to: '/leads?status=DISQUALIFIED', icon: 'xCircle' },
        { key: 'exported', label: t('nav.transferredMuninn'), to: '/leads?status=EXPORTED_TO_MUNINN', icon: 'muninn' },
      ],
    },
    { key: 'duplicates', label: t('nav.duplicates'), to: '/duplicates', icon: 'duplicates' },
    { key: 'assignments', label: t('nav.assignments'), to: '/assignments', icon: 'assignments' },
    { key: 'followups', label: t('nav.followups'), to: '/followups', icon: 'followups' },
    { key: 'work-queue', label: t('nav.workQueue'), to: '/work-queue', icon: 'layers' },
     { key: 'reports', label: t('nav.contactReports'), to: '/reports', icon: 'reports' },
     { key: 'agents', label: t('nav.agents'), to: '/agents', icon: 'agents' },
     { key: 'projects', label: t('nav.projects'), to: '/projects', icon: 'project' },
     { key: 'campaigns', label: t('nav.campaigns'), to: '/campaigns', icon: 'target' },
    {
      label: t('nav.settings'),
      children: [
        { key: 'settings-users', label: t('nav.users'), to: '/settings/users', icon: 'users' },
        { key: 'settings-roles', label: t('nav.roles'), to: '/settings/users?tab=roles', icon: 'roles' },
        { key: 'settings-scoring', label: t('nav.scoring'), to: '/settings/scoring', icon: 'scoring' },
        { key: 'settings-qualification', label: t('nav.qualification'), to: '/settings/qualification', icon: 'qualification' },
        { key: 'settings-llm', label: t('nav.llm'), to: '/settings/llm', icon: 'llm' },
        { key: 'settings-muninn', label: t('nav.muninn'), to: '/settings/muninn', icon: 'muninn' },
        { key: 'settings-system', label: t('nav.system'), to: '/settings/system', icon: 'system' },
      ],
    },
  ]
  if (can('audit.view')) {
    items.push({ key: 'audit', label: t('nav.audit'), to: '/audit', icon: 'audit' })
  }
  if (can('leads.history')) {
    items.push({ key: 'change-history', label: t('nav.changeHistory'), to: '/change-history', icon: 'layers' })
  }
  return items
}

export function Sidebar({
  collapsed,
  onToggle,
  mobileOpen,
  onMobileClose,
}: {
  collapsed: boolean
  onToggle: () => void
  mobileOpen: boolean
  onMobileClose: () => void
}) {
  const { t, fa } = useI18n()
  const navItems = useNavItems()
  const { pathname, search } = useLocation()

  const leafTargets = ['/dashboard']
  for (const item of navItems) {
    if (isGroup(item)) {
      for (const c of item.children) leafTargets.push(c.to)
    } else {
      leafTargets.push(item.to)
    }
  }
  const queryTargets = leafTargets.filter((to) => to.includes('?')).map((to) => `?${to.split('?')[1]}`)
  const active = (to: string) => isNavActive(to, pathname, search, queryTargets)

  const content = (
    <>
      <div className="sidebar-brand">
        <img src="/raven.png" alt="Huginn" className="brand-logo" />
        <div className="flex flex-col">
          <span className="brand-name">{t('app.name')}</span>
          <span className="brand-tag">{fa ? 'سیستم اطلاعات فروش' : 'Sales Intelligence'}</span>
        </div>
      </div>
      <nav className="sidebar-nav" aria-label={t('nav.dashboard')}>
        <Link
          to="/dashboard"
          className={`nav-item ${active('/dashboard') ? 'active' : ''}`.trim()}
          aria-current={active('/dashboard') ? 'page' : undefined}
          onClick={onMobileClose}
        >
          <Icon name="dashboard" />
          <span>{t('nav.dashboard')}</span>
        </Link>

        {navItems.map((item, i) => {
          if (isGroup(item)) {
            return (
              <div key={i}>
                <div className="nav-group-label">{item.label}</div>
                {item.children.map((c) => (
                  <Link
                    key={c.key}
                    to={c.to}
                    className={`nav-item ${active(c.to) ? 'active' : ''}`.trim()}
                    aria-current={active(c.to) ? 'page' : undefined}
                    onClick={onMobileClose}
                  >
                    <Icon name={c.icon} />
                    <span>{c.label}</span>
                  </Link>
                ))}
              </div>
            )
          }
          return (
            <Link
              key={item.key}
              to={item.to}
              className={`nav-item ${active(item.to) ? 'active' : ''}`.trim()}
              aria-current={active(item.to) ? 'page' : undefined}
              onClick={onMobileClose}
            >
              <Icon name={item.icon} />
              <span>{item.label}</span>
            </Link>
          )
        })}
      </nav>
      <div className="sidebar-footer">
        <button type="button" className="sidebar-collapse-btn" onClick={onToggle} aria-label={collapsed ? t('common.more') : t('common.close')}>
          <Icon name={collapsed ? 'expand' : 'collapse'} />
        </button>
      </div>
    </>
  )

  return (
    <>
      <div className={`sidebar-backdrop ${mobileOpen ? 'open' : ''}`} onClick={onMobileClose} aria-hidden="true" />
      <aside
        className={`sidebar ${collapsed ? 'sidebar-collapsed' : ''} ${mobileOpen ? '' : 'sidebar-drawer-closed'}`.trim()}
        aria-label={t('nav.settings')}
        dir={fa ? 'rtl' : 'ltr'}
      >
        {content}
      </aside>
    </>
  )
}
