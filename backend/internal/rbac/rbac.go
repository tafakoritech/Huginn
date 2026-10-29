// Package rbac defines roles and permission checks shared across the codebase.
package rbac

// Role is a user's authorization role.
type Role string

const (
	RoleSuperAdmin      Role = "super_admin"
	RoleSalesDevManager Role = "sales_development_manager"
	RoleSalesAgent      Role = "sales_agent"
	RoleViewer          Role = "viewer"
)

// Permissions.
const (
	PermUsersManage     = "users.manage"
	PermImportsCreate   = "imports.create"
	PermLeadsCreate     = "leads.create"
	PermLeadsEdit       = "leads.edit"
	PermLeadsAssign     = "leads.assign"
	PermLeadsMerge      = "leads.merge"
	PermLeadsQualify    = "leads.qualify"
	PermLeadsDisqualify = "leads.disqualify"
	PermLeadsExport     = "leads.export"
	PermLeadsContact    = "leads.contact"
	PermLeadsHistory    = "leads.history"
	PermReportsOverride = "reports.override"
	PermDashboardView   = "dashboard.view"
	PermDashboardMine   = "dashboard.mine"
	PermAuditView       = "audit.view"
	PermProjectsManage  = "projects.manage"
)

var rolePermissions = map[Role]map[string]bool{
	RoleSuperAdmin: {
		PermUsersManage:     true,
		PermImportsCreate:   true,
		PermLeadsCreate:     true,
		PermLeadsEdit:       true,
		PermLeadsAssign:     true,
		PermLeadsMerge:      true,
		PermLeadsQualify:    true,
		PermLeadsDisqualify: true,
		PermLeadsExport:     true,
		PermLeadsContact:    true,
		PermLeadsHistory:    true,
		PermReportsOverride: true,
		PermDashboardView:   true,
		PermDashboardMine:   true,
		PermAuditView:       true,
		PermProjectsManage:  true,
	},
	RoleSalesDevManager: {
		PermImportsCreate:   true,
		PermLeadsCreate:     true,
		PermLeadsEdit:       true,
		PermLeadsAssign:     true,
		PermLeadsMerge:      true,
		PermLeadsQualify:    true,
		PermLeadsDisqualify: true,
		PermLeadsExport:     true,
		PermLeadsContact:    true,
		PermLeadsHistory:    true,
		PermReportsOverride: true,
		PermDashboardView:   true,
		PermDashboardMine:   true,
		PermProjectsManage:  true,
	},
	RoleSalesAgent: {
		PermLeadsContact:  true,
		PermLeadsEdit:     true,
		PermDashboardMine: true,
	},
	RoleViewer: {
		PermDashboardView: true,
	},
}

// HasPermission reports whether role permits the given permission.
func HasPermission(role Role, permission string) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[permission]
}

// IsRole reports whether the string is a valid role.
func IsRole(r Role) bool {
	_, ok := rolePermissions[r]
	return ok
}

// AllPermissions returns every known permission in a stable order.
func AllPermissions() []string {
	return []string{
		PermUsersManage,
		PermImportsCreate,
		PermLeadsCreate,
		PermLeadsEdit,
		PermLeadsAssign,
		PermLeadsMerge,
		PermLeadsQualify,
		PermLeadsDisqualify,
		PermLeadsExport,
		PermLeadsContact,
		PermLeadsHistory,
		PermReportsOverride,
		PermDashboardView,
		PermDashboardMine,
		PermAuditView,
		PermProjectsManage,
	}
}

// StaticRoleDefinitions returns the built-in roles with their real permission
// lists. Used to seed the roles table so the dashboard reflects the actual
// authorization model (never mock data).
func StaticRoleDefinitions() map[Role][]string {
	out := map[Role][]string{}
	for role, perms := range rolePermissions {
		list := make([]string, 0, len(perms))
		for _, p := range AllPermissions() {
			if perms[p] {
				list = append(list, p)
			}
		}
		out[role] = list
	}
	return out
}
