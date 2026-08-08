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
	"cairn.cn/internal/httpx"
)

// AuthLogic 认证相关的业务逻辑
type AuthLogic struct {
	userRepo    auth.UserRepo
	sessionRepo auth.SessionRepo
}

func NewAuthLogic(userRepo auth.UserRepo, sessionRepo auth.SessionRepo) *AuthLogic {
	return &AuthLogic{userRepo: userRepo, sessionRepo: sessionRepo}
}

// Register 注册新用户。
func (l *AuthLogic) Register(ctx context.Context, username, password string) (httpx.Code, error) {
	username = strings.TrimSpace(username)

	if len(username) < 2 || len(username) > 64 {
		return httpx.CodeBadRequest, httpx.NewAppError("用户名需要 2–64 个字符", nil)
	}
	if len(password) < 6 {
		return httpx.CodeBadRequest, httpx.NewAppError("密码至少 6 个字符", nil)
	}

	if existing, _ := l.userRepo.GetByUsername(ctx, username); existing != nil {
		return httpx.CodeValidationFailed, httpx.NewAppError("用户名已被注册", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return httpx.CodeInternalError, httpx.NewAppError("密码加密失败", err)
	}

	u := &auth.User{
		Username: username,
		Password: string(hash),
		Role:     auth.RoleUser,
	}
	if err := l.userRepo.Create(ctx, u); err != nil {
		return httpx.CodeInternalError, httpx.NewAppError("创建用户失败", err)
	}
	return httpx.CodeCreated, nil
}

// LoginResult holds the product of a successful login.
type LoginResult struct {
	Token string
	User  *auth.User
}

// Login 验证用户名密码，创建 session.
func (l *AuthLogic) Login(ctx context.Context, username, password string) (LoginResult, httpx.Code, error) {
	username = strings.TrimSpace(username)

	if username == "" || password == "" {
		return LoginResult{}, httpx.CodeBadRequest, httpx.NewAppError("用户名和密码不能为空", nil)
	}

	u, err := l.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return LoginResult{}, httpx.CodeUnauthorized, httpx.NewAppError("用户名或密码错误", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return LoginResult{}, httpx.CodeUnauthorized, httpx.NewAppError("用户名或密码错误", err)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return LoginResult{}, httpx.CodeInternalError, httpx.NewAppError("生成会话失败", err)
	}
	token := hex.EncodeToString(tokenBytes)

	session := &auth.Session{
		Token:     token,
		UserID:    u.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := l.sessionRepo.Create(ctx, session); err != nil {
		return LoginResult{}, httpx.CodeInternalError, httpx.NewAppError("创建会话失败", err)
	}

	return LoginResult{Token: token, User: u}, httpx.CodeSuccess, nil
}

// Logout 登出，删除 session.
func (l *AuthLogic) Logout(ctx context.Context, token string) (httpx.Code, error) {
	if err := l.sessionRepo.Delete(ctx, token); err != nil {
		return httpx.CodeInternalError, httpx.NewAppError("登出失败", err)
	}
	return httpx.CodeNoContent, nil
}

// GetUserByToken 通过 session token 获取用户（内部方法，供中间件使用）。
func (l *AuthLogic) GetUserByToken(ctx context.Context, token string) (*auth.User, error) {
	session, err := l.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = l.sessionRepo.Delete(ctx, token)
		return nil, errors.New("会话已过期")
	}
	return l.userRepo.GetByID(ctx, session.UserID)
}
