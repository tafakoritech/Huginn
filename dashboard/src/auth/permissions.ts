import type { Role } from '@/api/types'

export const PERMISSIONS = {
  usersManage: 'users.manage',
  importsCreate: 'imports.create',
  leadsCreate: 'leads.create',
  leadsEdit: 'leads.edit',
  leadsAssign: 'leads.assign',
  leadsMerge: 'leads.merge',
  leadsQualify: 'leads.qualify',
  leadsDisqualify: 'leads.disqualify',
  leadsExport: 'leads.export',
   leadsContact: 'leads.contact',
   leadsHistory: 'leads.history',
   reportsOverride: 'reports.override',
   dashboardView: 'dashboard.view',
   dashboardMine: 'dashboard.mine',
   projectsManage: 'projects.manage',
  auditView: 'audit.view',
} as const

export type Permission = (typeof PERMISSIONS)[keyof typeof PERMISSIONS]

export const rolePermissions: Record<Role, string[]> = {
   super_admin: Object.values(PERMISSIONS),
   sales_development_manager: [
     PERMISSIONS.importsCreate,
     PERMISSIONS.leadsCreate,
     PERMISSIONS.leadsEdit,
     PERMISSIONS.leadsAssign,
     PERMISSIONS.leadsMerge,
     PERMISSIONS.leadsQualify,
     PERMISSIONS.leadsDisqualify,
     PERMISSIONS.leadsExport,
     PERMISSIONS.leadsContact,
     PERMISSIONS.leadsHistory,
     PERMISSIONS.reportsOverride,
     PERMISSIONS.dashboardView,
     PERMISSIONS.dashboardMine,
     PERMISSIONS.projectsManage,
   ],
  sales_agent: [PERMISSIONS.leadsContact, PERMISSIONS.leadsEdit, PERMISSIONS.dashboardMine],
  viewer: [PERMISSIONS.dashboardView],
}

export function hasPermission(role: Role, permission: string): boolean {
  return rolePermissions[role]?.includes(permission) ?? false
}
