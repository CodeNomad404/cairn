package httpx

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
)

// FileUpload contains bounded metadata followed by a single streamed file.
// File must be the last part, so callers can authorize metadata before reading it.
type FileUpload struct {
	Fields    map[string]string
	File      *multipart.Part
	reader    *multipart.Reader
	body      *uploadBody
	fileBytes int64
}

type uploadBody struct {
	io.ReadCloser
	bytes int64
}

func (b *uploadBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.bytes += int64(n)
	return n, err
}
func (u *FileUpload) Read(p []byte) (int, error) {
	n, err := u.File.Read(p)
	u.fileBytes += int64(n)
	return n, err
}

func ParseFileUpload(w http.ResponseWriter, r *http.Request, maxFileBytes int64, fields ...string) (*FileUpload, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileBytes+(64<<10))
	body := &uploadBody{ReadCloser: r.Body}
	r.Body = body
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, NewAppError("需要 multipart/form-data", err)
	}
	upload := &FileUpload{Fields: make(map[string]string), reader: reader, body: body}
	allowed := make(map[string]bool, len(fields))
	for _, field := range fields {
		allowed[field] = true
	}
	for {
		part, err := reader.NextRawPart()
		if err != nil {
			return nil, NewAppError("缺少上传文件", err)
		}
		name := part.FormName()
		if name == "file" && part.FileName() != "" {
			if len(upload.Fields) != len(allowed) {
				return nil, NewAppError("文件前必须提供完整元数据", nil)
			}
			upload.File = part
			if body.bytes > 64<<10 {
				return nil, NewAppError("上传元数据不能超过 64 KiB", nil)
			}
			return upload, nil
		}
		if !allowed[name] || part.FileName() != "" {
			return nil, NewAppError("上传字段不正确", nil)
		}
		if _, exists := upload.Fields[name]; exists {
			return nil, NewAppError("上传字段重复", nil)
		}
		value, err := io.ReadAll(io.LimitReader(part, 1025))
		if err != nil || len(value) > 1024 {
			return nil, NewAppError("上传字段过长或读取失败", err)
		}
		upload.Fields[name] = string(value)
	}
}

func (u *FileUpload) PositiveInt64(field string) (int64, error) {
	value, err := strconv.ParseInt(u.Fields[field], 10, 64)
	if err != nil || value < 1 {
		return 0, NewAppError(field+" must be positive", err)
	}
	return value, nil
}

// Finish rejects extra files, fields, or an incomplete closing boundary.
func (u *FileUpload) Finish() error {
	_, err := u.reader.NextRawPart()
	if errors.Is(err, io.EOF) {
		if _, err = io.Copy(io.Discard, u.body); err != nil {
			return NewAppError("上传请求过大或不完整", err)
		}
		if u.body.bytes-u.fileBytes > 64<<10 {
			return NewAppError("上传额外数据不能超过 64 KiB", nil)
		}
		return nil
	}
	return NewAppError("只允许一个文件且文件必须为最后一个字段", err)
}
