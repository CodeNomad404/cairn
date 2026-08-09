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
}

/** 前端自行组装树后的评论节点（API 返回的是扁平列表） */
export type CommentWithReplies = Comment & {
  replies: CommentWithReplies[]
}

export type CommentList = {
  items: Comment[]
}

export type CreateCommentInput = {
  content: string
  parent_id: number | null
}
