package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Validatable 由请求 DTO 实现，用于定义各自的 API 参数契约。
type Validatable interface {
	Validate() error
}

// Pagination 包含经过校验的分页查询参数。
type Pagination struct {
	Page     int
	PageSize int
}

// ParseBody 将 JSON 请求体解码并校验为类型 T 的值。
// 它强制 Body 大小上限、拒绝未知字段和尾随数据，并调用解码结果的 Validate。
// 失败时返回 AppError，其 Cause 为底层的解码/校验错误；它从不写响应，
// 由调用方决定 HTTP 状态码和响应体。
func ParseBody[T Validatable](w http.ResponseWriter, r *http.Request, maxBytes int64) (*T, error) {
	var req T
	if err := decodeJSON(w, r, &req, maxBytes); err != nil {
		return nil, NewAppError(err.Error(), err)
	}
	if err := req.Validate(); err != nil {
		return nil, NewAppError(err.Error(), err)
	}
	return &req, nil
}

// ParsePath 返回 chi 路径参数去除首尾空白后的非空值。
func ParsePath(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(chi.URLParam(r, name))
	if value == "" {
		return "", NewAppError("参数错误", nil)
	}
	return value, nil
}

// ParsePositiveInt64Path 解析一个正的 int64 路径参数。
func ParsePositiveInt64Path(r *http.Request, name string) (int64, error) {
	value, err := ParsePath(r, name)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, NewAppError("参数错误", err)
	}
	return id, nil
}

// ParsePagination 解析 page 和 page_size，应用标准默认值
// （page=1，page_size=20）和取值范围（page>=1，1<=page_size<=100）。
func ParsePagination(r *http.Request) (Pagination, error) {
	page, err := queryInt(r, "page", 1, 1, 0)
	if err != nil {
		return Pagination{}, err
	}
	pageSize, err := queryInt(r, "page_size", 20, 1, 100)
	if err != nil {
		return Pagination{}, err
	}
	return Pagination{Page: page, PageSize: pageSize}, nil
}

func queryInt(r *http.Request, name string, defaultValue, minValue, maxValue int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || maxValue > 0 && value > maxValue {
		return 0, NewAppError("参数错误", err)
	}
	return value, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return errors.New("request body must contain a single JSON value")
}
