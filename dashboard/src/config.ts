const env = import.meta.env as Record<string, string | undefined>

export const config = {
  apiBaseUrl: (env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1').replace(/\/$/, ''),
  appName: env.VITE_APP_NAME ?? 'Huginn',
  defaultLocale: (env.VITE_DEFAULT_LOCALE as 'fa' | 'en') ?? 'fa',
  defaultTheme: (env.VITE_DEFAULT_THEME as 'light' | 'dark') ?? 'light',
  useMockApi: env.VITE_USE_MOCK_API === 'true',
  storageKeys: {
    locale: 'huginn.locale',
    theme: 'huginn.theme',
    accessToken: 'huginn.access',
    refreshToken: 'huginn.refresh',
    user: 'huginn.user',
    sidebar: 'huginn.sidebar',
  },
}
