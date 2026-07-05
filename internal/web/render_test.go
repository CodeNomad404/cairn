package web

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicPageMetadata(t *testing.T) {
	renderer, err := NewRenderer("../../web/templates/*.html", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	data := struct{ PageData }{PublicPageData("测试标题", "测试描述", "/tools")}
	renderer.Render(recorder, "tools", data)
	body := recorder.Body.String()

	for _, want := range []string{
		`<meta name="baidu-site-verification" content="codeva-vz0JbMJvVI">`,
		`<meta name="description" content="测试描述">`,
		`<meta name="robots" content="index,follow">`,
		`<link rel="canonical" href="https://cairn.cn/tools">`,
		`<meta property="og:url" content="https://cairn.cn/tools">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered page does not contain %q", want)
		}
	}
}

func TestPrivatePageMetadata(t *testing.T) {
	renderer, err := NewRenderer("../../web/templates/*.html", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	data := struct {
		PageData
		Error string
	}{PageData: PrivatePageData("登录")}
	renderer.Render(recorder, "login", data)
	body := recorder.Body.String()

	if !strings.Contains(body, `<meta name="robots" content="noindex,nofollow">`) {
		t.Fatal("private page is missing noindex")
	}
	if strings.Contains(body, `rel="canonical"`) {
		t.Fatal("private page must not emit a canonical URL")
	}
}
