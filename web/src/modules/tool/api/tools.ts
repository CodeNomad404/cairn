import { request } from '../../../shared/api/http'

export type FormatJSONResult = {
  formatted: string
}

export function formatJSON(input: string) {
  return request<FormatJSONResult>('/api/v1/tools/json/format', {
    method: 'POST',
    body: JSON.stringify({ input }),
  })
}
