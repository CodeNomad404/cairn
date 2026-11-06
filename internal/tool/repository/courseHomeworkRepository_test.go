package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Each run migrates a new isolated schema in an explicitly configured test DB.
func homeworkTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("CAIRN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CAIRN_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("homework_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close(); _, _ = admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); admin.Close() })
	files, err := filepath.Glob("../../../migrations/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migration %s: %v", path, err)
		}
	}
	return db
}

func TestHomeworkRepositoryIsolationBindingAndSnapshotTransactions(t *testing.T) {
	db := homeworkTestDB(t)
	ctx := context.Background()
	if _, err := db.Exec(ctx, `INSERT INTO users (id,username,password) VALUES (7,'homework-user-seven','test'),(8,'homework-user-eight','test')`); err != nil {
		t.Fatal(err)
	}
	creds := NewCourseRegistrationCredentialsRepo(db)
	for _, id := range []int64{7, 8} {
		c := &toolmodel.CourseRegistrationCredentials{UserID: id, StudentID: fmt.Sprint(id), BJTUPassword: "old", TujianUsername: "keep", TujianPassword: "keep-secret"}
		if err := creds.Upsert(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	r := NewHomeworkRepo(db)
	course := toolmodel.HomeworkCourse{ID: "101", Name: "课程"}
	for _, id := range []int64{7, 8} {
		if err := r.Prepare(ctx, id, 1, "2026202701", []toolmodel.HomeworkCourse{course}); err != nil {
			t.Fatal(err)
		}
		items := []toolmodel.Homework{{UserID: id, BindingVersion: 1, TermCode: "2026202701", CourseID: "101", PlatformID: 99, Title: fmt.Sprint("user-", id)}}
		if err := r.SaveCourse(ctx, id, 1, "2026202701", course, items, ""); err != nil {
			t.Fatal(err)
		}
	}
	s, err := r.Snapshot(ctx, 7)
	if err != nil || len(s.Items) != 1 || s.Items[0].Title != "user-7" {
		t.Fatalf("snapshot=%+v error=%v", s, err)
	}
	id := s.Items[0].ID
	if _, err = r.Get(ctx, 8, id); !errors.Is(err, tool.ErrHomeworkNotFound) {
		t.Fatal("cross-user read allowed")
	}
	if err = r.SaveCourse(ctx, 7, 1, "2026202701", course, nil, "upstream unavailable"); err != nil {
		t.Fatal(err)
	}
	s, _ = r.Snapshot(ctx, 7)
	if len(s.Items) != 1 || s.Courses[0].Error == "" || s.Courses[0].LastSuccessAt == nil {
		t.Fatal("failed fetch erased last successful snapshot")
	}
	if err = r.SaveCourse(ctx, 7, 1, "2026202701", course, []toolmodel.Homework{{UserID: 7, BindingVersion: 1, CourseID: "101", PlatformID: 99, Title: "updated"}}, ""); err != nil {
		t.Fatal(err)
	}
	s, _ = r.Snapshot(ctx, 7)
	if s.Items[0].ID != id {
		t.Fatal("refresh changed internal homework ID")
	}
	if err = r.SaveCourse(ctx, 8, 1, "2026202701", course, []toolmodel.Homework{}, ""); err != nil {
		t.Fatal(err)
	}
	s8, _ := r.Snapshot(ctx, 8)
	if len(s8.Items) != 0 {
		t.Fatal("successful empty response did not clear course")
	}
	c := &toolmodel.CourseRegistrationCredentials{UserID: 7, StudentID: "new", BJTUPassword: "new"}
	if err = creds.BindMIS(ctx, c); err != nil {
		t.Fatal(err)
	}
	saved, _ := creds.GetByUserID(ctx, 7)
	if saved.TujianUsername != "keep" || saved.TujianPassword != "keep-secret" || saved.BindingVersion != 2 {
		t.Fatal("binding did not preserve Tujian or increment version")
	}
	s, _ = r.Snapshot(ctx, 7)
	if s.HasSnapshot || len(s.Items) != 0 || len(s.Courses) != 0 {
		t.Fatal("old account data visible after binding change")
	}
	if _, err = r.Get(ctx, 7, id); !errors.Is(err, tool.ErrHomeworkNotFound) {
		t.Fatal("old account details still visible")
	}
	if err = r.SaveCourse(ctx, 7, 1, "2026202701", course, nil, ""); !errors.Is(err, tool.ErrMISBindingChanged) {
		t.Fatal("old in-flight sync could write after binding change")
	}
	if err = r.Prepare(ctx, 7, 1, "2026202701", nil); !errors.Is(err, tool.ErrMISBindingChanged) {
		t.Fatal("old binding prepared a new run")
	}
	if err = r.Prepare(ctx, 7, 2, "2026202701", nil); err != nil {
		t.Fatal(err)
	}
	if err = r.SaveCourse(ctx, 7, 2, "2026202701", toolmodel.HomeworkCourse{}, nil, ""); err != nil {
		t.Fatal(err)
	}
	s, _ = r.Snapshot(ctx, 7)
	if !s.HasSnapshot || s.LastSuccessAt == nil || len(s.Items) != 0 {
		t.Fatal("zero-course successful run not represented")
	}
	// Exercise down/up against only this test's isolated schema.
	for _, suffix := range []string{"down", "up"} {
		sql, err := os.ReadFile("../../../migrations/000013_create_course_homework." + suffix + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
}
