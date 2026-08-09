export type PostSummary = {
  id: number
  slug: string
  title: string
  published_at: string | null
}

export type PostDetail = PostSummary & {
  content_html: string
}

export type PostList = {
  items: PostSummary[]
  page: number
  page_size: number
  has_more: boolean
}

export type CreatePostInput = {
  slug: string
  title: string
  content_md: string
}

export type Comment = {
  id: number
  post_id: number
  parent_id: number | null
  user_id: number
  content: string
  status: string
  created_at: string
  username: string
  replies: Comment[]
}

export type CommentList = {
  items: Comment[]
}

export type CreateCommentInput = {
  content: string
  parent_id: number | null
}
