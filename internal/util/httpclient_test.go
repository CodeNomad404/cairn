package util

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewCookieClient(t *testing.T) {
	client, err := NewCookieClient(5 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if client.Jar == nil {
		t.Fatal("cookie jar is nil")
	}
	if client.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v", client.Timeout)
	}
}

func TestReadResponseBody(t *testing.T) {
	response := &http.Response{Body: io.NopCloser(strings.NewReader("hello"))}
	data, err := ReadResponseBody(response, 5)
	if err != nil || string(data) != "hello" {
		t.Fatalf("data = %q, err = %v", data, err)
	}

	response = &http.Response{Body: io.NopCloser(strings.NewReader("too long"))}
	if _, err := ReadResponseBody(response, 3); err == nil {
		t.Fatal("oversized response error = nil")
	}
}
