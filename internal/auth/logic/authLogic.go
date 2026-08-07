package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
)

// Credentials is the request body for login and register.
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Validate checks the API contract for credentials.
func (c Credentials) Validate() error {
	if strings.TrimSpace(c.Username) == "" || c.Password == "" {
		return errors.New("username and password are required")
	}
	return nil
}

// AuthLogic 认证相关的业务逻辑
type AuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AuthLogic {
	return &AuthLogic{ctx: ctx, svcCtx: svcCtx}
}

// Register 注册新用户，返回创建的用户（不含密码）
func (l *AuthLogic) Register(req *Credentials) (any, response.Code, string) {
	username := strings.TrimSpace(req.Username)
	password := req.Password

	if len(username) < 2 || len(username) > 64 {
		return nil, response.CodeBadRequest, "用户名需要 2–64 个字符"
	}
	if len(password) < 6 {
		return nil, response.CodeBadRequest, "密码至少 6 个字符"
	}

	if existing, _ := l.svcCtx.UserRepo.GetByUsername(l.ctx, username); existing != nil {
		return nil, response.CodeValidationFailed, "用户名已被注册"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, response.CodeInternalError, "密码加密失败"
	}

	u := &auth.User{
		Username: username,
		Password: string(hash),
		Role:     auth.RoleUser,
	}
	if err := l.svcCtx.UserRepo.Create(l.ctx, u); err != nil {
		return nil, response.CodeInternalError, "创建用户失败"
	}
	return u, response.CodeSuccess, ""
}

// LoginResult holds the product of a successful login.
type LoginResult struct {
	Token string
	User  *auth.User
}

// Login 验证用户名密码，创建 session.
func (l *AuthLogic) Login(req *Credentials) (any, response.Code, string) {
	username := strings.TrimSpace(req.Username)
	password := req.Password

	if username == "" || password == "" {
		return nil, response.CodeBadRequest, "用户名和密码不能为空"
	}

	u, err := l.svcCtx.UserRepo.GetByUsername(l.ctx, username)
	if err != nil {
		return nil, response.CodeUnauthorized, "用户名或密码错误"
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return nil, response.CodeUnauthorized, "用户名或密码错误"
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, response.CodeInternalError, "生成会话失败"
	}
	token := hex.EncodeToString(tokenBytes)

	session := &auth.Session{
		Token:     token,
		UserID:    u.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := l.svcCtx.SessionRepo.Create(l.ctx, session); err != nil {
		return nil, response.CodeInternalError, "创建会话失败"
	}

	return &LoginResult{Token: token, User: u}, response.CodeSuccess, ""
}

// Logout 登出，删除 session.
func (l *AuthLogic) Logout(token string) (any, response.Code, string) {
	if err := l.svcCtx.SessionRepo.Delete(l.ctx, token); err != nil {
		return nil, response.CodeInternalError, "登出失败"
	}
	return nil, response.CodeNoContent, ""
}

// GetUserByToken 通过 session token 获取用户（内部方法，供中间件使用）。
func (l *AuthLogic) GetUserByToken(token string) (*auth.User, error) {
	session, err := l.svcCtx.SessionRepo.GetByToken(l.ctx, token)
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = l.svcCtx.SessionRepo.Delete(l.ctx, token)
		return nil, errors.New("会话已过期")
	}
	return l.svcCtx.UserRepo.GetByID(l.ctx, session.UserID)
}
