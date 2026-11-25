package logic

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type transferUploadRepo struct {
	tool.TransferRepo
	item      toolmodel.TransferItem
	abandoned bool
}

func (r *transferUploadRepo) CreateItem(_ context.Context, _ int64, item *toolmodel.TransferItem) error {
	item.ID = 1
	item.CreatedAt = time.Now()
	r.item = *item
	return nil
}
func (r *transferUploadRepo) Activate(_ context.Context, _ int64, size int64) error {
	r.item.State = "active"
	r.item.SizeBytes = size
	return nil
}
func (r *transferUploadRepo) Abandon(context.Context, int64) error { r.abandoned = true; return nil }
func (r *transferUploadRepo) GetItem(context.Context, int64, int64) (*toolmodel.TransferItem, error) {
	return &r.item, nil
}

type transferZeroReader struct{}

func (transferZeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestTransferFileSizeBoundaryAndFailedUploads(t *testing.T) {
	tests := []struct {
		name        string
		size        int64
		finishError bool
		want        httpx.Code
	}{
		{"empty", 0, false, httpx.CodeCreated},
		{"exactly 100 MiB", tool.TransferFileLimit, false, httpx.CodeCreated},
		{"one byte too large", tool.TransferFileLimit + 1, false, httpx.CodeBadRequest},
		{"invalid trailing boundary", 10, true, httpx.CodeBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &transferUploadRepo{}
			directory := t.TempDir()
			logic := NewTransferLogic(repo, directory, slog.New(slog.NewTextHandler(io.Discard, nil)))
			result, code, err := logic.Upload(context.Background(), 7, 1, "文件.bin", io.LimitReader(transferZeroReader{}, tt.size), func() error {
				if tt.finishError {
					return httpx.NewAppError("bad boundary", errors.New("test"))
				}
				return nil
			})
			if code != tt.want {
				t.Fatalf("code=%d error=%v", code, err)
			}
			if code == httpx.CodeCreated {
				if err != nil || repo.abandoned {
					t.Fatal("successful upload abandoned", err)
				}
				data := result.(TransferItemResult)
				if data.SizeBytes != tt.size {
					t.Fatal("wrong file size")
				}
				if repo.item.StorageKey == "文件.bin" || len(repo.item.StorageKey) != 32 {
					t.Fatal("unsafe storage key")
				}
				entries, _ := os.ReadDir(directory)
				if len(entries) != 1 || strings.HasSuffix(entries[0].Name(), ".part") {
					t.Fatal("temporary file not promoted")
				}
			} else if !repo.abandoned {
				t.Fatal("failed upload not marked for cleanup")
			}
		})
	}
}
