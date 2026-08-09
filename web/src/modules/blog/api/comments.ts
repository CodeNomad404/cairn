import { request } from '../../../shared/api/http'
import type { CommentList, Comment, CreateCommentInput } from '../types'

export function fetchComments(slug: string) {
  return request<CommentList>(`/api/v1/posts/${slug}/comments`)
}

export function createComment(slug: string, input: CreateCommentInput) {
  return request<Comment>(`/api/v1/posts/${slug}/comments`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function deleteComment(id: number) {
  return request<void>(`/api/v1/comments/${id}`, { method: 'DELETE' })
}
