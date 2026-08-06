package logic

import (
	"encoding/json"
)

// JSONLogic JSON 格式化工具的业务逻辑
type JSONLogic struct{}

// NewJSONLogic 创建 JSONLogic
func NewJSONLogic() *JSONLogic {
	return &JSONLogic{}
}

// Format 校验并格式化 JSON
func (s *JSONLogic) Format(input string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Validate 校验 JSON 是否合法
func (s *JSONLogic) Validate(input string) bool {
	return json.Valid([]byte(input))
}
