package util

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// NewCookieClient 创建能够在连续请求间保存 Cookie 的 HTTP 客户端。
func NewCookieClient(timeout time.Duration) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &http.Client{Jar: jar, Timeout: timeout}, nil
}

// ReadResponseBody 读取有限大小的响应体，超过限制时返回错误。
// 调用方仍负责关闭 response.Body。
func ReadResponseBody(response *http.Response, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxBytes)
	}
	return data, nil
}
