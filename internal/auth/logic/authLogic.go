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
	authmodel "cairn.cn/internal/auth/model"
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

// Register 原子地创建新用户及其会话。
func (l *AuthLogic) Register(ctx context.Context, username, password string) (LoginResult, httpx.Code, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return LoginResult{}, httpx.CodeInternalError, httpx.NewAppError("密码加密失败", err)
	}
	u := &authmodel.User{
		Username: strings.TrimSpace(username),
		Password: string(hash),
		Role:     auth.RoleUser,
	}
	session, err := newSession(0)
	if err != nil {
		return LoginResult{}, httpx.CodeInternalError, httpx.NewAppError("生成会话失败", err)
	}
	if err := l.userRepo.CreateWithSession(ctx, u, session); err != nil {
		if errors.Is(err, auth.ErrUsernameTaken) {
			return LoginResult{}, httpx.CodeValidationFailed, httpx.NewAppError("用户名已被注册", nil)
		}
		return LoginResult{}, httpx.CodeInternalError, httpx.NewAppError("注册暂未完成，请稍后重试", err)
	}
	return LoginResult{Token: session.Token, User: toUserResult(u)}, httpx.CodeCreated, nil
}

// UserResult 是对外公开的用户信息。
type UserResult struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginResult 保存成功登录的产物。
type LoginResult struct {
	Token string
	User  UserResult
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

	session, err := newSession(u.ID)
	if err != nil {
		return LoginResult{}, httpx.CodeInternalError, httpx.NewAppError("生成会话失败", err)
	}
	if err := l.sessionRepo.Create(ctx, session); err != nil {
		return LoginResult{}, httpx.CodeInternalError, httpx.NewAppError("创建会话失败", err)
	}

	return LoginResult{Token: session.Token, User: toUserResult(u)}, httpx.CodeSuccess, nil
}

func newSession(userID int64) (*authmodel.Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	now := time.Now()
	return &authmodel.Session{
		Token:     hex.EncodeToString(tokenBytes),
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}, nil
}

// Logout 登出，删除 session.
func (l *AuthLogic) Logout(ctx context.Context, token string) (httpx.Code, error) {
	if err := l.sessionRepo.Delete(ctx, token); err != nil {
		return httpx.CodeInternalError, httpx.NewAppError("登出失败", err)
	}
	return httpx.CodeNoContent, nil
}

// GetPrincipalByToken 通过 session token 获取当前请求身份（内部方法，供中间件使用）。
func (l *AuthLogic) GetPrincipalByToken(ctx context.Context, token string) (*auth.Principal, error) {
	session, err := l.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = l.sessionRepo.Delete(ctx, token)
		return nil, errors.New("会话已过期")
	}
	user, err := l.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	return &auth.Principal{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

// Me 返回当前登录用户对外可见的信息。
func (l *AuthLogic) Me(ctx context.Context, principal *auth.Principal) (UserResult, httpx.Code, error) {
	if principal == nil {
		return UserResult{}, httpx.CodeUnauthorized, httpx.NewAppError("login required", nil)
	}
	user, err := l.userRepo.GetByID(ctx, principal.ID)
	if err != nil {
		return UserResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to load user", err)
	}
	return toUserResult(user), httpx.CodeSuccess, nil
}

func toUserResult(user *authmodel.User) UserResult {
	return UserResult{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}
