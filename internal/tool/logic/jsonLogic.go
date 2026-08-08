package logic

import (
	"encoding/json"

	"cairn.cn/internal/util"
)

// JSONLogic JSON 格式化工具的业务逻辑
type JSONLogic struct{}

// NewJSONLogic 创建 JSONLogic
func NewJSONLogic() *JSONLogic {
	return &JSONLogic{}
}

// Format 校验并格式化 JSON
func (l *JSONLogic) Format(input string) (any, util.Code, string) {
	var v any
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return nil, util.CodeBadRequest, "invalid JSON: " + err.Error()
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, util.CodeInternalError, "failed to format JSON"
	}
	return map[string]string{"formatted": string(out)}, util.CodeSuccess, ""
}

// Validate 校验 JSON 是否合法
func (l *JSONLogic) Validate(input string) bool {
	return json.Valid([]byte(input))
}
