import { createBrowserRouter, Navigate, Outlet } from 'react-router-dom'
import { AuthProvider } from '@/auth/AuthContext'
import { AppShell } from '@/components/layout/AppShell'
import { ProtectedRoute } from '@/auth/ProtectedRoute'
import { LoginPage } from '@/pages/LoginPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { LeadsPage } from '@/pages/leads/LeadsPage'
import { LeadDetailPage } from '@/pages/leads/LeadDetailPage'
import { QualifiedPage } from '@/pages/leads/QualifiedPage'
import { ImportsPage } from '@/pages/imports/ImportsPage'
import { ImportNewPage } from '@/pages/imports/ImportNewPage'
import { ImportDetailPage } from '@/pages/imports/ImportDetailPage'
import { DocumentsPage } from '@/pages/DocumentsPage'
import { DocumentDetailPage } from '@/pages/DocumentDetailPage'
import { DuplicatesPage } from '@/pages/DuplicatesPage'
import { AssignmentsPage } from '@/pages/AssignmentsPage'
import { AgentsPage, AgentDetailPage } from '@/pages/AgentsPage'
import { ReportsPage } from '@/pages/reports/ReportsPage'
import { ReportDetailPage } from '@/pages/reports/ReportDetailPage'
import { FollowupsPage } from '@/pages/FollowupsPage'
import { WorkQueuePage } from '@/pages/WorkQueuePage'
import { ProjectsPage } from '@/pages/projects/ProjectsPage'
import { ProjectDetailPage } from '@/pages/projects/ProjectDetailPage'
import { CampaignsPage, CampaignDetailPage } from '@/pages/campaigns/CampaignsPage'
import { UsersPage } from '@/pages/settings/UsersPage'
import {
  ScoringSettingsPage,
  QualificationSettingsPage,
  LLMSettingsPage,
  MuninnSettingsPage,
  SystemSettingsPage,
} from '@/pages/settings/SettingsPages'
import { AuditPage } from '@/pages/AuditPage'
import { ChangeHistoryPage } from '@/pages/ChangeHistoryPage'
import { NotFoundPage } from '@/pages/NotFoundPage'

export const router = createBrowserRouter([
  {
    // AuthProvider lives inside the Router so useNavigate() is always valid.
    element: (
      <AuthProvider>
        <Outlet />
      </AuthProvider>
    ),
    children: [
      { path: '/login', element: <LoginPage /> },
      {
        element: <ProtectedRoute />,
        children: [
          {
            element: <AppShell />,
            children: [
          { path: '/', element: <Navigate to="/dashboard" replace /> },
          { path: '/dashboard', element: <DashboardPage /> },
          { path: '/imports', element: <ImportsPage /> },
          { path: '/imports/new', element: <ImportNewPage /> },
          { path: '/imports/:id', element: <ImportDetailPage /> },
          { path: '/documents', element: <DocumentsPage /> },
          { path: '/documents/:id', element: <DocumentDetailPage /> },
          { path: '/leads', element: <LeadsPage /> },
          { path: '/leads/qualified', element: <QualifiedPage /> },
          { path: '/leads/:id', element: <LeadDetailPage /> },
          { path: '/duplicates', element: <DuplicatesPage /> },
          { path: '/assignments', element: <AssignmentsPage /> },
          { path: '/agents', element: <AgentsPage /> },
          { path: '/agents/:id', element: <AgentDetailPage /> },
          { path: '/reports', element: <ReportsPage /> },
          { path: '/reports/:id', element: <ReportDetailPage /> },
           { path: '/followups', element: <FollowupsPage /> },
           { path: '/work-queue', element: <WorkQueuePage /> },
           { path: '/projects', element: <ProjectsPage /> },
           { path: '/projects/:id', element: <ProjectDetailPage /> },
           { path: '/campaigns', element: <CampaignsPage /> },
           { path: '/campaigns/:id', element: <CampaignDetailPage /> },
          { path: '/settings/users', element: <UsersPage /> },
          { path: '/settings/scoring', element: <ScoringSettingsPage /> },
          { path: '/settings/qualification', element: <QualificationSettingsPage /> },
          { path: '/settings/llm', element: <LLMSettingsPage /> },
          { path: '/settings/muninn', element: <MuninnSettingsPage /> },
          { path: '/settings/system', element: <SystemSettingsPage /> },
          { path: '/audit', element: <AuditPage /> },
          { path: '/change-history', element: <ChangeHistoryPage /> },
          { path: '*', element: <NotFoundPage /> },
            ],
          },
        ],
      },
    ],
  },
])
