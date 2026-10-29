import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/api'
import { setOnUnauthorized, setTokensForClient, tokenStore } from '@/api/client'
import { config } from '@/config'
import type { AuthUser } from '@/api/types'
import { PERMISSIONS, hasPermission } from './permissions'

export { PERMISSIONS }
export type { Permission } from './permissions'

type AuthContextValue = {
  user: AuthUser | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
  can: (permission: string) => boolean
  isPrivileged: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

function readStoredUser(): AuthUser | null {
  try {
    const raw = localStorage.getItem(config.storageKeys.user)
    return raw ? (JSON.parse(raw) as AuthUser) : null
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(() => readStoredUser())
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const logout = useCallback(
    async (reason?: string) => {
      try {
        await api.logout()
      } catch {
        /* idempotent */
      }
      tokenStore.clear()
      setUser(null)
      if (reason !== 'session_expired') navigate('/login')
    },
    [navigate],
  )

  useEffect(() => {
    setOnUnauthorized((reason) => {
      void logout(reason)
    })
    return () => setOnUnauthorized(null)
  }, [logout])

  const login = useCallback(async (username: string, password: string) => {
    setLoading(true)
    try {
      const result = await api.login(username, password)
      setTokensForClient(result.access_token, result.refresh_token)
      localStorage.setItem(config.storageKeys.user, JSON.stringify(result.user))
      setUser(result.user)
    } finally {
      setLoading(false)
    }
  }, [])

  const refreshUser = useCallback(async () => {
    try {
      const me = await api.me()
      setUser(me)
      localStorage.setItem(config.storageKeys.user, JSON.stringify(me))
    } catch {
      /* keep existing session */
    }
  }, [])

  const can = useCallback(
    (permission: string) => {
      if (!user) return false
      return hasPermission(user.role, permission)
    },
    [user],
  )

  const isPrivileged = useMemo(
    () => user?.role === 'super_admin' || user?.role === 'sales_development_manager',
    [user],
  )

  const value = useMemo(
    () => ({ user, loading, login, logout, refreshUser, can, isPrivileged }),
    [user, loading, login, logout, refreshUser, can, isPrivileged],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
