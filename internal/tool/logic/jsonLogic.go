package logic

import (
	"context"
	"encoding/json"

	"cairn.cn/internal/svc"
	"cairn.cn/internal/util"
)

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
