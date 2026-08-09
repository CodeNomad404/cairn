import { request } from '../../../shared/api/http'
import type { PostList, PostDetail, CreatePostInput } from '../types'

export function fetchPosts(page = 1, pageSize = 20) {
  return request<PostList>(`/api/v1/posts?page=${page}&page_size=${pageSize}`)
}

export function fetchPost(slug: string) {
  return request<PostDetail>(`/api/v1/posts/${slug}`)
}

export function createPost(input: CreatePostInput) {
  return request<PostDetail>('/api/v1/posts', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function deletePost(slug: string) {
  return request<void>(`/api/v1/posts/${slug}`, { method: 'DELETE' })
}
