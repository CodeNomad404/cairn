package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/tool"
	toolhandler "cairn.cn/internal/tool/handler"
	toollogic "cairn.cn/internal/tool/logic"
	toolmodel "cairn.cn/internal/tool/model"
	"github.com/go-chi/chi/v5"
)

func TestTransferPersistenceIsolationExpiryAndFiles(t *testing.T) {
	db := homeworkTestDB(t)
	ctx := context.Background()
	// Verify the new migration can be rolled back and reapplied in an empty schema.
	for _, direction := range []string{"down", "up"} {
		sql, err := os.ReadFile("../../../migrations/000014_create_transfer." + direction + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `INSERT INTO users(id,username,password,role) VALUES(7,'transfer-a','test','admin'),(8,'transfer-b','test','admin')`); err != nil {
		t.Fatal(err)
	}
	repo := NewTransferRepo(db)
	directory := t.TempDir()
	logic := toollogic.NewTransferLogic(repo, directory, slog.New(slog.NewTextHandler(io.Discard, nil)))
	router := chi.NewRouter()
	toolhandler.NewTransferHandler(logic).RegisterRoutes(router)
	call := func(user int64, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/tool/"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(authmid.WithPrincipal(req.Context(), &auth.Principal{ID: user, Role: auth.RoleAdmin}))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	first, err := repo.CreateChannel(ctx, 7, "命令")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.CreateChannel(ctx, 7, "命令"); !errors.Is(err, tool.ErrTransferDuplicate) {
		t.Fatal("duplicate accepted", err)
	}
	if _, err = repo.CreateChannel(ctx, 8, "命令"); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RenameChannel(ctx, 8, first.ID, "other"); !errors.Is(err, tool.ErrTransferNotFound) {
		t.Fatal("cross-account rename", err)
	}
	original := "  echo \"你好\"\r\n\tWrite-Output 'C:\\work' `\n  next  "
	input, _ := json.Marshal(map[string]any{"channel_id": first.ID, "text": original})
	response := call(7, "POST", "transfer/create", string(input))
	if response.Code != 201 {
		t.Fatal(response.Code, response.Body.String())
	}
	var created struct {
		Data toollogic.TransferItemResult `json:"data"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Data.Text == nil || *created.Data.Text != original {
		t.Fatal("text was changed")
	}
	if created.Data.ExpiresAt.Sub(created.Data.CreatedAt) != 15*24*time.Hour {
		t.Fatal("wrong expiry")
	}
	response = call(7, "GET", fmt.Sprintf("transfer/get?id=%d", created.Data.ID), "")
	if response.Code != 200 {
		t.Fatal("second device cannot read", response.Body.String())
	}
	response = call(8, "GET", fmt.Sprintf("transfer/get?id=%d", created.Data.ID), "")
	if response.Code != 404 {
		t.Fatal("cross-account read", response.Code)
	}
	response = call(8, "POST", "transfer/create", string(input))
	if response.Code != 404 {
		t.Fatal("cross-account write", response.Code)
	}
	response = call(8, "POST", "transfer/delete", fmt.Sprintf(`{"id":%d}`, created.Data.ID))
	if response.Code != 404 {
		t.Fatal("cross-account delete", response.Code)
	}

	fileBytes := bytes.Repeat([]byte{0, 1, 2, 127, 255}, 1000)
	upload := func() toollogic.TransferItemResult {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("channel_id", fmt.Sprint(first.ID))
		part, e := writer.CreateFormFile("file", "实习资料.bin")
		if e != nil {
			t.Fatal(e)
		}
		_, _ = part.Write(fileBytes)
		_ = writer.Close()
		req := httptest.NewRequest("POST", "/tool/transfer/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = req.WithContext(authmid.WithPrincipal(ctx, &auth.Principal{ID: 7, Role: auth.RoleAdmin}))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatal(w.Code, w.Body.String())
		}
		var result struct {
			Data toollogic.TransferItemResult `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &result)
		return result.Data
	}
	file := upload()
	sameName := upload()
	if sameName.ID == file.ID {
		t.Fatal("same name overwrote file")
	}
	response = call(7, "GET", fmt.Sprintf("transfer/download?id=%d", file.ID), "")
	if response.Code != 200 || !bytes.Equal(response.Body.Bytes(), fileBytes) {
		t.Fatal("download differs")
	}
	if !strings.HasPrefix(response.Header().Get("Content-Disposition"), "attachment;") || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("unsafe download headers")
	}
	response = call(8, "GET", fmt.Sprintf("transfer/download?id=%d", file.ID), "")
	if response.Code != 404 {
		t.Fatal("cross-account download", response.Code)
	}
	diskItem, err := repo.GetItem(ctx, 7, file.ID)
	if err != nil {
		t.Fatal(err)
	}
	restarted := toollogic.NewTransferLogic(NewTransferRepo(db), directory, slog.Default())
	download, _, err := restarted.Download(ctx, 7, file.ID)
	if err != nil {
		t.Fatal("restart lost file", err)
	}
	_ = download.File.Close()

	// Expiration is enforced at read time even before the cleanup worker runs.
	if _, err = db.Exec(ctx, `UPDATE transfer_items SET created_at=NOW()-INTERVAL '361 hours' WHERE id=ANY($1)`, []int64{file.ID, created.Data.ID}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{file.ID, created.Data.ID} {
		response = call(7, "GET", fmt.Sprintf("transfer/get?id=%d", id), "")
		if response.Code != 404 {
			t.Fatal("expired item readable")
		}
	}
	if response = call(7, "GET", fmt.Sprintf("transfer/download?id=%d", file.ID), ""); response.Code != 404 {
		t.Fatal("expired file downloadable")
	}
	if err = logic.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(directory, diskItem.StorageKey)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expired file still on disk", err)
	}
	if _, err = repo.GetItem(ctx, 7, sameName.ID); err != nil {
		t.Fatal("cleanup deleted fresh item", err)
	}
	channels, err := repo.ListChannels(ctx, 7)
	if err != nil || len(channels) != 1 || channels[0].FileCount != 1 || channels[0].TextCount != 0 {
		t.Fatal("channel/counts lost", channels, err)
	}

	// No aggregate quota: metadata beyond the former 1 GiB cap is accepted.
	for n := range 12 {
		i := &toolmodel.TransferItem{ChannelID: first.ID, Kind: "file", Filename: "large.bin", StorageKey: fmt.Sprintf("%032x", n+1), SizeBytes: tool.TransferFileLimit, State: "active"}
		if err = repo.CreateItem(ctx, 7, i); err != nil {
			t.Fatal("aggregate limit remains", err)
		}
	}
	used, err := repo.Usage(ctx, 7)
	if err != nil || used <= 1<<30 {
		t.Fatal("usage above 1 GiB rejected", used, err)
	}

	// A channel deleted during upload remains hidden; pending files are not purged
	// while a request may still be writing them.
	pending := &toolmodel.TransferItem{ChannelID: first.ID, Kind: "file", Filename: "pending.bin", StorageKey: strings.Repeat("f", 32), SizeBytes: tool.TransferFileLimit, State: "pending"}
	if err = repo.CreateItem(ctx, 7, pending); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteChannel(ctx, 7, first.ID); err != nil {
		t.Fatal(err)
	}
	if err = logic.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err = db.QueryRow(ctx, `SELECT state FROM transfer_items WHERE id=$1`, pending.ID).Scan(&state); err != nil || state != "pending" {
		t.Fatal("cleaned live upload", state, err)
	}
	if err = repo.Activate(ctx, pending.ID, 5); !errors.Is(err, tool.ErrTransferNotFound) {
		t.Fatal("activated deleted channel", err)
	}
	if err = repo.Abandon(ctx, pending.ID); err != nil {
		t.Fatal(err)
	}
	if err = logic.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	channels, err = repo.ListChannels(ctx, 7)
	if err != nil || len(channels) != 0 {
		t.Fatal("deleted channel remains", err)
	}
	var count int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM transfer_items`).Scan(&count); err != nil || count != 0 {
		t.Fatal("deleted contents remain", count, err)
	}
}

func TestTransferPaginationAndCleanupRetry(t *testing.T) {
	db := homeworkTestDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `INSERT INTO users(id,username,password) VALUES(7,'transfer-retry','test')`); err != nil {
		t.Fatal(err)
	}
	repo := NewTransferRepo(db)
	c, err := repo.CreateChannel(ctx, 7, "test")
	if err != nil {
		t.Fatal(err)
	}
	for range 21 {
		if err = repo.CreateItem(ctx, 7, &toolmodel.TransferItem{ChannelID: c.ID, Kind: "text", Body: "text", SizeBytes: 4, State: "active"}); err != nil {
			t.Fatal(err)
		}
	}
	items, err := repo.ListItems(ctx, 7, c.ID, "text", 1, 20)
	if err != nil || len(items) != 21 {
		t.Fatal("pagination lookahead", len(items), err)
	}
	items, err = repo.ListItems(ctx, 7, c.ID, "text", 2, 20)
	if err != nil || len(items) != 1 {
		t.Fatal("page 2", len(items), err)
	}
	items, err = repo.ListItems(ctx, 7, c.ID, "file", 1, 20)
	if err != nil || len(items) != 0 {
		t.Fatal("filter", err)
	}
	directory := t.TempDir()
	logic := toollogic.NewTransferLogic(repo, directory, slog.New(slog.NewTextHandler(io.Discard, nil)))
	key := strings.Repeat("a", 32)
	file := &toolmodel.TransferItem{ChannelID: c.ID, Kind: "file", Filename: "retry.bin", StorageKey: key, SizeBytes: 0, State: "active"}
	if err = repo.CreateItem(ctx, 7, file); err != nil {
		t.Fatal(err)
	}
	// A nonempty directory reliably simulates an undeletable file on Windows/Linux.
	blocked := filepath.Join(directory, key)
	if err = os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(blocked, "block")
	if err = os.WriteFile(child, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = repo.DeleteItem(ctx, 7, file.ID); err != nil {
		t.Fatal(err)
	}
	if err = logic.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err = db.QueryRow(ctx, `SELECT state FROM transfer_items WHERE id=$1`, file.ID).Scan(&state); err != nil || state != "deleting" {
		t.Fatal("lost retry marker", err)
	}
	if err = os.Remove(child); err != nil {
		t.Fatal(err)
	}
	if err = logic.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(blocked); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("retry did not remove file")
	}
	var count int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM transfer_items WHERE id=$1`, file.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("retry did not remove row", err)
	}
	// A full batch of failed deletes must not block newer cleanup candidates.
	if _, err = db.Exec(ctx, `INSERT INTO transfer_items(channel_id,kind,body,size_bytes,state)
 SELECT $1,'text','x',1,'deleting' FROM generate_series(1,201)`, c.ID); err != nil {
		t.Fatal(err)
	}
	first, err := repo.CleanupItems(ctx)
	if err != nil || len(first) != 200 {
		t.Fatal("first cleanup batch", len(first), err)
	}
	seen := map[int64]bool{}
	for _, item := range first {
		seen[item.ID] = true
	}
	second, err := repo.CleanupItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	progress := false
	for _, item := range second {
		if !seen[item.ID] {
			progress = true
		}
	}
	if !progress {
		t.Fatal("failed cleanup batch starved later records")
	}
}
