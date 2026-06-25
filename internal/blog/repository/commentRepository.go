package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/blog"
)

// CommentRepository 负责 comments 表的数据库操作
type CommentRepository struct {
	db *pgxpool.Pool
}

// NewCommentRepo 创建 CommentRepository 实例
func NewCommentRepo(db *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 插入一条评论
func (r *CommentRepository) Create(ctx context.Context, c *blog.Comment) error {
	sql := `INSERT INTO comments (post_id, parent_id, user_id, content)
	        VALUES ($1, $2, $3, $4)
	        RETURNING id, status, created_at`
	return r.db.QueryRow(ctx, sql,
		c.PostID, c.ParentID, c.UserID, c.Content,
	).Scan(&c.ID, &c.Status, &c.CreatedAt)
}

// GetByID 根据 id 查询一条评论
func (r *CommentRepository) GetByID(ctx context.Context, id int64) (*blog.Comment, error) {
	c := &blog.Comment{}
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.post_id, c.parent_id, c.user_id, u.username,
		       c.content, c.status, c.created_at
		FROM comments AS c
		JOIN users AS u ON u.id = c.user_id
		WHERE c.id = $1`, id).Scan(
		&c.ID, &c.PostID, &c.ParentID, &c.UserID, &c.Username,
		&c.Content, &c.Status, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ListByPostID 返回某篇文章的顶层评论（含 replies）
func (r *CommentRepository) ListByPostID(ctx context.Context, postID int64) ([]blog.Comment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.post_id, c.parent_id, c.user_id, u.username,
		       c.content, c.status, c.created_at
		FROM comments AS c
		JOIN users AS u ON u.id = c.user_id
		WHERE c.post_id = $1 AND c.status = 'approved'
		ORDER BY c.created_at ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []blog.Comment
	for rows.Next() {
		var c blog.Comment
		err := rows.Scan(&c.ID, &c.PostID, &c.ParentID, &c.UserID, &c.Username,
			&c.Content, &c.Status, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		all = append(all, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 组装树形：顶层评论 + 嵌套 replies
	return buildTree(all), nil
}

// Delete 软删除评论（设为 deleted 状态）
func (r *CommentRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE comments SET status = 'deleted' WHERE id = $1`, id)
	return err
}

// buildTree 将平铺的评论列表转成树形（1 级回复）
func buildTree(all []blog.Comment) []blog.Comment {
	byID := make(map[int64]*blog.Comment, len(all))

	for i := range all {
		byID[all[i].ID] = &all[i]
	}

	// 先建立所有父子关系
	for i := range all {
		c := &all[i]
		if c.ParentID != nil {
			if parent, ok := byID[*c.ParentID]; ok {
				parent.Replies = append(parent.Replies, *c)
			}
		}
	}

	// 再收集顶层评论（此时 replies 已填充完毕）
	var roots []blog.Comment
	for i := range all {
		if all[i].ParentID == nil {
			roots = append(roots, all[i])
		}
	}

	if roots == nil {
		roots = []blog.Comment{}
	}
	return roots
}
