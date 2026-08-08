package logic

import (
	"encoding/json"

	"cairn.cn/internal/httpx"
)

// JSONLogic JSON 格式化工具的业务逻辑
type JSONLogic struct{}

type FormatJSONResult struct {
	Formatted string `json:"formatted"`
}

// NewJSONLogic 创建 JSONLogic
func NewJSONLogic() *JSONLogic {
	return &JSONLogic{}
}

// Format 校验并格式化 JSON
func (l *JSONLogic) Format(input string) (FormatJSONResult, httpx.Code, error) {
	var v any
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return FormatJSONResult{}, httpx.CodeBadRequest, httpx.NewAppError("invalid JSON: "+err.Error(), err)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return FormatJSONResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to format JSON", err)
	}
	return FormatJSONResult{Formatted: string(out)}, httpx.CodeSuccess, nil
}

// Validate 校验 JSON 是否合法
func (l *JSONLogic) Validate(input string) bool {
	return json.Valid([]byte(input))
}
