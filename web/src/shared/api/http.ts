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

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(path, {
    ...init,
    headers,
    credentials: 'include',
  })
  if (response.status === 204) {
    return undefined as T
  }

  const body: APIEnvelope | null = await response.json().catch(() => null)
  if (!response.ok || (body && body.code !== 0 && body.code !== 1)) {
    throw new APIRequestError(
      body?.message ?? '请求失败，请稍后重试。',
      response.status,
      body?.code ?? -1,
    )
  }
  return (body?.data ?? body) as T
}
