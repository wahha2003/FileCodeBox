const trimTrailingSlash = (value: string) => value.replace(/\/+$/, '')

const readEnvUrl = (value: unknown): string | undefined => {
  if (typeof value !== 'string') {
    return undefined
  }

  const trimmed = value.trim()
  return trimmed ? trimTrailingSlash(trimmed) : undefined
}

const isLocalHost = (hostname: string) =>
  hostname === 'localhost' || hostname.endsWith('.localhost')

const inferApiBaseUrl = () => {
  const configured = readEnvUrl(import.meta.env.VITE_API_BASE_URL)
  if (configured) {
    return configured
  }

  if (typeof window === 'undefined') {
    return 'http://api.localhost:12345'
  }

  const { protocol, hostname } = window.location

  if (import.meta.env.DEV || isLocalHost(hostname)) {
    return 'http://api.localhost:12345'
  }

  if (hostname.startsWith('api.')) {
    return trimTrailingSlash(window.location.origin)
  }

  const normalizedHost = hostname.replace(/^(www|app)\./i, '')
  return `${protocol}//api.${normalizedHost}`
}

const inferPublicBaseUrl = () => {
  const configured =
    readEnvUrl(import.meta.env.VITE_PUBLIC_BASE_URL) ??
    readEnvUrl(import.meta.env.VITE_SITE_BASE_URL)
  if (configured) {
    return configured
  }

  if (typeof window === 'undefined') {
    return 'http://localhost:3000'
  }

  const { protocol, hostname, origin } = window.location

  if (hostname === 'api.localhost') {
    return 'http://localhost:3000'
  }

  if (hostname.startsWith('api.')) {
    return `${protocol}//${hostname.slice(4)}`
  }

  return trimTrailingSlash(origin)
}

export const API_BASE_URL = inferApiBaseUrl()
export const PUBLIC_BASE_URL = inferPublicBaseUrl()

export const buildApiUrl = (path: string) =>
  new URL(path.startsWith('/') ? path.slice(1) : path, `${API_BASE_URL}/`).toString()

export const buildPublicUrl = (path = '/') =>
  new URL(path.startsWith('/') ? path.slice(1) : path, `${PUBLIC_BASE_URL}/`).toString()

export const buildPublicShareUrl = (code: string) =>
  `${trimTrailingSlash(PUBLIC_BASE_URL)}/#/share/${encodeURIComponent(code)}`

