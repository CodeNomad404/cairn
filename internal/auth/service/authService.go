package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"cairn.cn/internal/auth"
)

// AuthService 认证相关的业务逻辑
type AuthService struct {
	userRepo    auth.UserRepo
	sessionRepo auth.SessionRepo
}

func New(userRepo auth.UserRepo, sessionRepo auth.SessionRepo) *AuthService {
	return &AuthService{userRepo: userRepo, sessionRepo: sessionRepo}
}

// Register 注册新用户，返回创建的用户（不含密码）
func (s *AuthService) Register(ctx context.Context, username, password string) (*auth.User, error) {
	username = strings.TrimSpace(username)
	if len(username) < 2 || len(username) > 64 {
		return nil, errors.New("用户名需要 2–64 个字符")
	}
	if len(password) < 6 {
		return nil, errors.New("密码至少 6 个字符")
	}

	// 检查用户名是否已存在
	if existing, _ := s.userRepo.GetByUsername(ctx, username); existing != nil {
		return nil, errors.New("用户名已被注册")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	u := &auth.User{
		Username: username,
		Password: string(hash),
		Role:     "user", // 新注册默认普通用户
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, errors.New("创建用户失败")
	}
	return u, nil
}

// Login 验证用户名密码，创建 session，返回 token
func (s *AuthService) Login(ctx context.Context, username, password string) (string, *auth.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", nil, errors.New("用户名和密码不能为空")
	}

	u, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	// 生成随机 session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, errors.New("生成会话失败")
	}
	token := hex.EncodeToString(tokenBytes)

	session := &auth.Session{
		Token:     token,
		UserID:    u.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 天有效
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", nil, errors.New("创建会话失败")
	}

	return token, u, nil
}

// Logout 登出，删除 session
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessionRepo.Delete(ctx, token)
}

// GetUserByToken 通过 session token 获取用户
func (s *AuthService) GetUserByToken(ctx context.Context, token string) (*auth.User, error) {
	session, err := s.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = s.sessionRepo.Delete(ctx, token)
		return nil, errors.New("会话已过期")
	}
	return s.userRepo.GetByID(ctx, session.UserID)
}
