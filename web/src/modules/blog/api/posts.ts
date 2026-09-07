import { request } from '../../../shared/api/http'
import type { CreatePostRequest, PostDetailResult, PostListResult } from '../types'

export function fetchPosts(page = 1, pageSize = 20) {
  return request<PostListResult>('/api/v1/posts', {
    params: { page, page_size: pageSize },
  })
}

export function fetchPost(slug: string, signal?: AbortSignal) {
  return request<PostDetailResult>(`/api/v1/posts/${encodeURIComponent(slug)}`, { signal })
}

export function createPost(input: CreatePostRequest) {
  return request<PostDetailResult>('/api/v1/posts', {
    method: 'POST',
    json: input,
  })
}

export function deletePost(slug: string) {
  return request<void>(`/api/v1/posts/${encodeURIComponent(slug)}`, {
    method: 'DELETE'
  })
}
