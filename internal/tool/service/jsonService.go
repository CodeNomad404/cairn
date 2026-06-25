package service

import (
	"encoding/json"
)

// JSONService JSON 格式化工具的业务逻辑
type JSONService struct{}

// New 创建 JSONService
func New() *JSONService {
	return &JSONService{}
}

// Format 校验并格式化 JSON
func (s *JSONService) Format(input string) (string, error) {
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
func (s *JSONService) Validate(input string) bool {
	return json.Valid([]byte(input))
}
