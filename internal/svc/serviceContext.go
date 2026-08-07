package svc

import (
	"cairn.cn/internal/auth"
	"cairn.cn/internal/blog"
	"cairn.cn/internal/config"
)

// ServiceContext is the server-wide dependency container.
type ServiceContext struct {
	Config      *config.Config
	PostRepo    blog.PostRepo
	CommentRepo blog.CommentRepo
	UserRepo    auth.UserRepo
	SessionRepo auth.SessionRepo
}
