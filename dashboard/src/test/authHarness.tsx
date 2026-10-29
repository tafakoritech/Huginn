import { createContext, useContext, useMemo, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { hasPermission } from '@/auth/permissions'
import { api } from '@/api'
import { config } from '@/config'
import type { AuthUser } from '@/api/types'

export type AuthContextValue = {
  user: AuthUser | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
  can: (permission: string) => boolean
  isPrivileged: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ user, children }: { user: AuthUser | null; children: ReactNode }) {
  const navigate = useNavigate()
  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      loading: false,
      login: async (username: string, password: string) => {
        const r = await api.login(username, password)
        localStorage.setItem(config.storageKeys.accessToken, r.access_token)
        localStorage.setItem(config.storageKeys.refreshToken, r.refresh_token)
        localStorage.setItem(config.storageKeys.user, JSON.stringify(r.user))
        navigate('/dashboard')
      },
      logout: async () => {
        localStorage.removeItem(config.storageKeys.accessToken)
        navigate('/login')
      },
      refreshUser: async () => {},
      can: (p) => (user ? hasPermission(user.role, p) : false),
      isPrivileged: user?.role === 'super_admin' || user?.role === 'sales_development_manager',
    }),
    [user, navigate],
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
