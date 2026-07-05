package handler

import (
	"strings"
	"testing"
)

func TestPostDescription(t *testing.T) {
	description := postDescription("Go 模板渲染实践", `<h1>Go 模板渲染实践</h1><h2>摘要</h2><p>记录模板组织、数据传递与页面复用经验。</p><script>ignored</script>`)
	if description != "Go 模板渲染实践：记录模板组织、数据传递与页面复用经验。" {
		t.Fatalf("description = %q", description)
	}

	long := postDescription("长文", "<p>"+strings.Repeat("界", 200)+"</p>")
	if len([]rune(long)) != 155 || !strings.HasSuffix(long, "…") {
		t.Fatalf("long description length = %d, value = %q", len([]rune(long)), long)
	}
}
