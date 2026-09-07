import { request } from '../../../shared/api/http'
import type { CommentListResult, CommentResult, CreateCommentRequest } from '../types'

export function fetchComments(slug: string, signal?: AbortSignal) {
  return request<CommentListResult>(`/api/v1/posts/${encodeURIComponent(slug)}/comments`, { signal })
}

export function createComment(slug: string, input: CreateCommentRequest) {
  return request<CommentResult>(`/api/v1/posts/${encodeURIComponent(slug)}/comments`, {
    method: 'POST',
    json: input,
  })
}

export function deleteComment(id: number) {
  return request<void>(`/api/v1/comments/${encodeURIComponent(String(id))}`, { method: 'DELETE' })
}
