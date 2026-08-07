package logic

import (
	"context"
	"encoding/json"
	"strings"

	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
)

// FormatJSONRequest is the request body for the JSON formatter.
type FormatJSONRequest struct {
	Input string `json:"input"`
}

// Validate checks the API contract for formatting JSON.
func (r FormatJSONRequest) Validate() error {
	if strings.TrimSpace(r.Input) == "" {
		return nil // empty input is allowed; Format will handle it
	}
	return nil
}

// JSONLogic JSON 格式化工具的业务逻辑
type JSONLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewJSONLogic 创建 JSONLogic
func NewJSONLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JSONLogic {
	return &JSONLogic{ctx: ctx, svcCtx: svcCtx}
}

// Format 校验并格式化 JSON
func (l *JSONLogic) Format(input string) (any, response.Code, string) {
	var v any
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return nil, response.CodeBadRequest, "invalid JSON: " + err.Error()
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, response.CodeInternalError, "failed to format JSON"
	}
	return map[string]string{"formatted": string(out)}, response.CodeSuccess, ""
}

// Validate 校验 JSON 是否合法
func (l *JSONLogic) Validate(input string) bool {
	return json.Valid([]byte(input))
}
