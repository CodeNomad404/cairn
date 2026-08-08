package auth

// Principal 是当前请求的最小登录身份，仅用于认证与授权判断。
// 不包含密码哈希、Session Token 等数据库内部状态，也不承担 API 响应职责。
type Principal struct {
	ID       int64
	Username string
	Role     string
}
