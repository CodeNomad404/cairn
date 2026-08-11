export class APIRequestError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code: number,
  ) {
    super(message)
  }
}

type APIEnvelope = {
  code: number
  message?: string
  data?: unknown
}

function isAPIEnvelope(value: unknown): value is APIEnvelope {
  if (typeof value !== 'object' || value === null) return false
  const candidate = value as Record<string, unknown>
  return typeof candidate.code === 'number'
    && (candidate.message === undefined || typeof candidate.message === 'string')
}

export interface RequestOptions extends Omit<RequestInit, 'body'> {
  params?: Record<string, string | number>
  json?: unknown
  form?: FormData
  signal?: AbortSignal
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { params, json, form, signal, ...init } = options

  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')

  let body: BodyInit | undefined
  if (form) {
    body = form
  } else if (json !== undefined) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(json)
  }

  let url = path
  if (params) {
    const qs = new URLSearchParams()
    for (const [key, value] of Object.entries(params)) {
      qs.set(key, String(value))
    }
    url = `${path}?${qs.toString()}`
  }

  const response = await fetch(url, {
    ...init,
    headers,
    body,
    credentials: 'include',
    signal,
  })

  if (response.status === 204) {
    return undefined as T
  }

  const payload: unknown = await response.json().catch(() => null)
  const envelope = isAPIEnvelope(payload) ? payload : null

  if (!response.ok || !envelope || (envelope.code !== 0 && envelope.code !== 1)) {
    throw new APIRequestError(
      envelope?.message ?? '请求失败，请稍后重试。',
      response.status,
      envelope?.code ?? -1,
    )
  }

  return (envelope.data ?? envelope) as T
}
