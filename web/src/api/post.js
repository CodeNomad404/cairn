import { request } from './request'

// 获取帖子列表
export function getPostList() {
  return request('/api/post/list')
}

// 发布帖子
export function createPost(data) {
  return request('/api/post', {
    method: 'POST',
    body: JSON.stringify(data)
  })
}
