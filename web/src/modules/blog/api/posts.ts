import { request } from '../../../shared/api/http'
import type { PostList, PostDetail, CommentList, Comment, CreateCommentInput } from '../types'

export function fetchPosts(page = 1, pageSize = 20) {
  return request<PostList>(`/api/v1/posts?page=${page}&page_size=${pageSize}`)
}

export function fetchPost(slug: string) {
  return request<PostDetail>(`/api/v1/posts/${slug}`)
}
