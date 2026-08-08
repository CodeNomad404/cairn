package model

import (
	"reflect"
	"testing"
)

// TestInternalModelsDeclareNoJSONTags 验证内部模型不声明 JSON API 契约。
// 注意：没有 JSON 标签不等于不能被 json.Marshal 序列化；
// 对外字段只能通过 Logic 中的 XxxResult 声明。
func TestInternalModelsDeclareNoJSONTags(t *testing.T) {
	models := []any{User{}, Session{}}
	for _, model := range models {
		typ := reflect.TypeOf(model)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if tag := field.Tag.Get("json"); tag != "" {
				t.Errorf("%s.%s 声明了 json 标签 %q，内部模型不应承担 JSON API 契约",
					typ.Name(), field.Name, tag)
			}
		}
	}
}
