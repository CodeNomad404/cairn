export type PostSummaryResult = {
  id: number
  slug: string
  title: string
  published_at: string | null
}

export type PostDetailResult = PostSummaryResult & {
  content_html: string
}

export type PostListResult = {
  items: PostSummaryResult[]
  page: number
  page_size: number
  has_more: boolean
}

export type CreatePostRequest = {
  slug: string
  title: string
  content_md: string
}

export type CommentResult = {
  id: number
  post_id: number
  parent_id: number | null
  user_id: number
  content: string
  status: string
  created_at: string
  username: string
  replies?: CommentResult[]
}

export type CommentListResult = {
  items: CommentResult[]
}

export type CreateCommentRequest = {
  content: string
  parent_id: number | null
}
