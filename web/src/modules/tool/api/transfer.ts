import { APIRequestError, request } from '../../../shared/api/http'

export interface TransferChannel {
  id: number
  name: string
  created_at: string
  updated_at: string
  file_count: number
  text_count: number
}

export interface TransferItem {
  id: number
  channel_id: number
  kind: 'text' | 'file'
  text?: string
  summary?: string
  filename: string
  size_bytes: number
  created_at: string
  expires_at: string
}

export interface TransferUsage {
  used_bytes: number
  retention_days: number
  file_limit_bytes: number
  text_limit_bytes: number
}

export type TransferKind = '' | 'text' | 'file'
const base = '/api/v1/tool/'

export function listTransferChannels() {
  return request<{ items: TransferChannel[] }>(base + 'transfer-channel/list')
}
export function createTransferChannel(name: string) {
  return request<TransferChannel>(base + 'transfer-channel/create', { method: 'POST', json: { name } })
}
export function renameTransferChannel(id: number, name: string) {
  return request<TransferChannel>(base + 'transfer-channel/update', { method: 'POST', json: { id, name } })
}
export function deleteTransferChannel(id: number) {
  return request<void>(base + 'transfer-channel/delete', { method: 'POST', json: { id } })
}
export function listTransferItems(channelID: number, page: number, kind: TransferKind) {
  return request<{ items: TransferItem[]; page: number; page_size: number; has_more: boolean }>(base + 'transfer/list', {
    params: { channel_id: channelID, page, page_size: 20, kind },
  })
}
export function getTransferItem(id: number) {
  return request<TransferItem>(base + 'transfer/get', { params: { id } })
}
export function createTransferText(channelID: number, text: string) {
  return request<TransferItem>(base + 'transfer/create', { method: 'POST', json: { channel_id: channelID, text } })
}
export function uploadTransferFile(channelID: number, file: File) {
  const form = new FormData()
  form.append('channel_id', String(channelID))
  form.append('file', file)
  return request<TransferItem>(base + 'transfer/upload', { method: 'POST', form })
}
export function deleteTransferItem(id: number) {
  return request<void>(base + 'transfer/delete', { method: 'POST', json: { id } })
}
export function getTransferUsage() {
  return request<TransferUsage>(base + 'transfer-usage/get')
}

export async function downloadTransferFile(item: TransferItem) {
  const response = await fetch(base + 'transfer/download?id=' + item.id, { credentials: 'include', cache: 'no-store' })
  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    throw new APIRequestError(payload?.message ?? '下载失败，请重试。', response.status, payload?.code ?? -1)
  }
  const url = URL.createObjectURL(await response.blob())
  const link = document.createElement('a')
  link.href = url
  link.download = item.filename
  document.body.append(link)
  link.click()
  link.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 60_000)
}
