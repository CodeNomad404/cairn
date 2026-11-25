package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type TransferLogic struct {
	repo      tool.TransferRepo
	directory string
	logger    *slog.Logger
}

func NewTransferLogic(repo tool.TransferRepo, directory string, logger *slog.Logger) *TransferLogic {
	return &TransferLogic{repo: repo, directory: directory, logger: logger}
}

type TransferChannelResult struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	FileCount int64     `json:"file_count"`
	TextCount int64     `json:"text_count"`
}
type TransferChannelListResult struct {
	Items []TransferChannelResult `json:"items"`
}
type TransferItemResult struct {
	ID        int64     `json:"id"`
	ChannelID int64     `json:"channel_id"`
	Kind      string    `json:"kind"`
	Text      *string   `json:"text,omitempty"`
	Summary   *string   `json:"summary,omitempty"`
	Filename  string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
type TransferItemListResult struct {
	Items    []TransferItemResult `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	HasMore  bool                 `json:"has_more"`
}
type TransferUsageResult struct {
	UsedBytes      int64 `json:"used_bytes"`
	RetentionDays  int   `json:"retention_days"`
	FileLimitBytes int64 `json:"file_limit_bytes"`
	TextLimitBytes int   `json:"text_limit_bytes"`
}
type TransferDownloadResult struct {
	File      *os.File
	Filename  string
	CreatedAt time.Time
}

func transferChannelResult(c *toolmodel.TransferChannel) TransferChannelResult {
	return TransferChannelResult{c.ID, c.Name, c.CreatedAt, c.UpdatedAt, c.FileCount, c.TextCount}
}
func transferItemResult(i *toolmodel.TransferItem, summary bool) TransferItemResult {
	result := TransferItemResult{ID: i.ID, ChannelID: i.ChannelID, Kind: i.Kind, Filename: i.Filename, SizeBytes: i.SizeBytes, CreatedAt: i.CreatedAt, ExpiresAt: i.CreatedAt.Add(tool.TransferRetentionDays * 24 * time.Hour)}
	if i.Kind == "text" {
		if summary {
			result.Summary = &i.Body
		} else {
			result.Text = &i.Body
		}
	}
	return result
}
func transferError(err error) (httpx.Code, error) {
	switch {
	case errors.Is(err, tool.ErrTransferNotFound):
		return httpx.CodeNotFound, httpx.NewAppError("通道或内容不存在", err)
	case errors.Is(err, tool.ErrTransferDuplicate):
		return httpx.CodeValidationFailed, httpx.NewAppError("通道名称已存在", err)
	default:
		return httpx.CodeInternalError, httpx.NewAppError("传输服务暂时不可用", err)
	}
}

func (l *TransferLogic) CreateChannel(ctx context.Context, userID int64, name string) (any, httpx.Code, error) {
	c, err := l.repo.CreateChannel(ctx, userID, strings.TrimSpace(name))
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	return transferChannelResult(c), httpx.CodeCreated, nil
}
func (l *TransferLogic) ListChannels(ctx context.Context, userID int64) (any, httpx.Code, error) {
	channels, err := l.repo.ListChannels(ctx, userID)
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	result := TransferChannelListResult{Items: make([]TransferChannelResult, 0, len(channels))}
	for i := range channels {
		result.Items = append(result.Items, transferChannelResult(&channels[i]))
	}
	return result, httpx.CodeSuccess, nil
}
func (l *TransferLogic) RenameChannel(ctx context.Context, userID, id int64, name string) (any, httpx.Code, error) {
	c, err := l.repo.RenameChannel(ctx, userID, id, strings.TrimSpace(name))
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	return transferChannelResult(c), httpx.CodeSuccess, nil
}
func (l *TransferLogic) DeleteChannel(ctx context.Context, userID, id int64) (httpx.Code, error) {
	if err := l.repo.DeleteChannel(ctx, userID, id); err != nil {
		return transferError(err)
	}
	return httpx.CodeNoContent, nil
}
func (l *TransferLogic) CreateText(ctx context.Context, userID, channelID int64, text string) (any, httpx.Code, error) {
	i := &toolmodel.TransferItem{ChannelID: channelID, Kind: "text", Body: text, SizeBytes: int64(len(text)), State: "active"}
	if err := l.repo.CreateItem(ctx, userID, i); err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	return transferItemResult(i, false), httpx.CodeCreated, nil
}

func (l *TransferLogic) Upload(ctx context.Context, userID, channelID int64, filename string, reader io.Reader, finish func() error) (any, httpx.Code, error) {
	filename = path.Base(strings.ReplaceAll(filename, "\\", "/"))
	if filename == "." || filename == ".." || filename == "/" || len(filename) > 255 || !utf8.ValidString(filename) || strings.IndexFunc(filename, unicode.IsControl) >= 0 {
		return nil, httpx.CodeBadRequest, httpx.NewAppError("文件名无效或超过 255 字节", nil)
	}
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	i := &toolmodel.TransferItem{ChannelID: channelID, Kind: "file", Filename: filename, StorageKey: hex.EncodeToString(key), State: "pending"}
	if err := l.repo.CreateItem(ctx, userID, i); err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	complete := false
	defer func() {
		if !complete {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := l.repo.Abandon(cleanupCtx, i.ID); err != nil {
				l.logger.Error("mark failed transfer for cleanup", "id", i.ID, "error", err)
			}
		}
	}()
	if err := os.MkdirAll(l.directory, 0700); err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	temporary := filepath.Join(l.directory, i.StorageKey+".part")
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	size, copyErr := io.Copy(file, io.LimitReader(reader, tool.TransferFileLimit+1))
	closeErr := file.Close()
	var writeError *os.PathError
	if errors.As(copyErr, &writeError) {
		code, err := transferError(copyErr)
		return nil, code, err
	}
	if copyErr != nil || size > tool.TransferFileLimit {
		return nil, httpx.CodeBadRequest, httpx.NewAppError("文件超过 100 MiB、上传中断或写入失败", copyErr)
	}
	if closeErr != nil {
		code, err := transferError(closeErr)
		return nil, code, err
	}
	if err = finish(); err != nil {
		return nil, httpx.CodeBadRequest, err
	}
	if err = ctx.Err(); err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	if err = os.Rename(temporary, filepath.Join(l.directory, i.StorageKey)); err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	if err = l.repo.Activate(ctx, i.ID, size); err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	complete = true
	// Activation starts the retention period after the upload has finished.
	saved, err := l.repo.GetItem(ctx, userID, i.ID)
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	return transferItemResult(saved, false), httpx.CodeCreated, nil
}

func (l *TransferLogic) ListItems(ctx context.Context, userID, channelID int64, kind string, page, pageSize int) (any, httpx.Code, error) {
	items, err := l.repo.ListItems(ctx, userID, channelID, kind, page, pageSize)
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	result := TransferItemListResult{Items: make([]TransferItemResult, 0), Page: page, PageSize: pageSize, HasMore: len(items) > pageSize}
	if result.HasMore {
		items = items[:pageSize]
	}
	for i := range items {
		result.Items = append(result.Items, transferItemResult(&items[i], true))
	}
	return result, httpx.CodeSuccess, nil
}
func (l *TransferLogic) GetItem(ctx context.Context, userID, id int64) (any, httpx.Code, error) {
	item, err := l.repo.GetItem(ctx, userID, id)
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	return transferItemResult(item, false), httpx.CodeSuccess, nil
}
func (l *TransferLogic) Download(ctx context.Context, userID, id int64) (*TransferDownloadResult, httpx.Code, error) {
	item, err := l.repo.GetItem(ctx, userID, id)
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	if item.Kind != "file" {
		return nil, httpx.CodeNotFound, httpx.NewAppError("文件不存在", nil)
	}
	file, err := os.Open(filepath.Join(l.directory, item.StorageKey))
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	return &TransferDownloadResult{File: file, Filename: item.Filename, CreatedAt: item.CreatedAt}, httpx.CodeSuccess, nil
}
func (l *TransferLogic) DeleteItem(ctx context.Context, userID, id int64) (httpx.Code, error) {
	if err := l.repo.DeleteItem(ctx, userID, id); err != nil {
		return transferError(err)
	}
	return httpx.CodeNoContent, nil
}
func (l *TransferLogic) Usage(ctx context.Context, userID int64) (any, httpx.Code, error) {
	size, err := l.repo.Usage(ctx, userID)
	if err != nil {
		code, err := transferError(err)
		return nil, code, err
	}
	return TransferUsageResult{size, tool.TransferRetentionDays, tool.TransferFileLimit, tool.TransferTextLimit}, httpx.CodeSuccess, nil
}

func (l *TransferLogic) Cleanup(ctx context.Context) error {
	items, err := l.repo.CleanupItems(ctx)
	if err != nil {
		return err
	}
	for _, i := range items {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if i.StorageKey != "" {
			for _, suffix := range []string{".part", ""} {
				err = os.Remove(filepath.Join(l.directory, i.StorageKey+suffix))
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					break
				}
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				l.logger.Error("remove transfer file", "id", i.ID, "error", err)
				continue
			}
		}
		if err = l.repo.Purge(ctx, i.ID); err != nil {
			return err
		}
	}
	// Also purges empty channels which have been manually deleted.
	return l.repo.Purge(ctx, 0)
}

func (l *TransferLogic) StartCleanup(ctx context.Context) error {
	if err := os.MkdirAll(l.directory, 0700); err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := l.Cleanup(cleanupCtx); err != nil && ctx.Err() == nil {
				l.logger.Error("transfer cleanup failed", "error", err)
			}
			cancel()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return nil
}
