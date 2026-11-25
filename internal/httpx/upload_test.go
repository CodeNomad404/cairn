package httpx

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func multipartTestBody(t *testing.T, fields [][2]string, files int) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, field := range fields {
		if err := writer.WriteField(field[0], field[1]); err != nil {
			t.Fatal(err)
		}
	}
	for range files {
		part, err := writer.CreateFormFile("file", "测试.bin")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte{0, 1, 2, 255})
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

func TestStreamedUploadContract(t *testing.T) {
	tests := []struct {
		name   string
		fields [][2]string
		files  int
		valid  bool
	}{
		{"valid", [][2]string{{"channel_id", "7"}}, 1, true},
		{"missing metadata", nil, 1, false},
		{"duplicate metadata", [][2]string{{"channel_id", "7"}, {"channel_id", "8"}}, 1, false},
		{"unknown field", [][2]string{{"other", "7"}}, 1, false},
		{"oversize field", [][2]string{{"channel_id", strings.Repeat("7", 1025)}}, 1, false},
		{"missing file", [][2]string{{"channel_id", "7"}}, 0, false},
		{"extra file", [][2]string{{"channel_id", "7"}}, 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := multipartTestBody(t, tt.fields, tt.files)
			request := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			request.Header.Set("Content-Type", contentType)
			upload, err := ParseFileUpload(httptest.NewRecorder(), request, 100, "channel_id")
			if err == nil {
				var got []byte
				got, err = io.ReadAll(upload)
				if err == nil {
					err = upload.Finish()
				}
				if tt.valid && !bytes.Equal(got, []byte{0, 1, 2, 255}) {
					t.Fatal("file changed")
				}
			}
			if (err == nil) != tt.valid {
				t.Fatalf("error=%v valid=%v", err, tt.valid)
			}
		})
	}
}

func TestUploadRejectsTruncatedBoundaryAndEpilogue(t *testing.T) {
	body, contentType := multipartTestBody(t, [][2]string{{"channel_id", "7"}}, 1)
	for _, data := range [][]byte{body[:len(body)-10], append(append([]byte(nil), body...), bytes.Repeat([]byte{'x'}, 70<<10)...)} {
		request := httptest.NewRequest("POST", "/", bytes.NewReader(data))
		request.Header.Set("Content-Type", contentType)
		upload, err := ParseFileUpload(httptest.NewRecorder(), request, 100, "channel_id")
		if err == nil {
			_, err = io.Copy(io.Discard, upload)
			if err == nil {
				err = upload.Finish()
			}
		}
		if err == nil {
			t.Fatal("accepted incomplete or oversized multipart request")
		}
	}
}
