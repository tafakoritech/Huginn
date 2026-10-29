import { config } from '@/config'
import { realApi } from './real'
import { mockApi } from '@/mocks/mockApi'
import type { ApiEndpoints } from './interface'

export const api: ApiEndpoints = config.useMockApi ? mockApi : realApi

export const isMockApi = config.useMockApi
