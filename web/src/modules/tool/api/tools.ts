import { request } from '../../../shared/api/http'
import type { FormatJSONRequest, FormatJSONResult } from '../types'

export function formatJSON(input: FormatJSONRequest) {
  return request<FormatJSONResult>('/api/v1/tools/json/format', {
    method: 'POST',
    json: input,
  })
}
