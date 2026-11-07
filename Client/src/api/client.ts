import createClient from 'openapi-fetch'
import type { paths } from './schema.d.ts'

export type { components, operations, paths } from './schema.d.ts'

export const apiBaseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '/api').replace(/\/$/, '')

export const api = createClient<paths>({
  baseUrl: apiBaseUrl,
  credentials: 'include',
})
