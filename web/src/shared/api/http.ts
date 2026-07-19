export class APIRequestError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: string,
  ) {
    super(message)
  }
}

type ErrorPayload = {
  error?: {
    code?: string
    message?: string
  }
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

  const body = await response.json().catch(() => null)
  if (!response.ok) {
    const error = body as ErrorPayload | null
    throw new APIRequestError(
      error?.error?.message ?? '请求失败，请稍后重试。',
      response.status,
      error?.error?.code,
    )
  }
  return body as T
}
