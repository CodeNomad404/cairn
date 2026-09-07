package tool

import (
	"context"
	"errors"
	"time"

	toolmodel "cairn.cn/internal/tool/model"
)

var ErrCourseRegistrationCredentialsNotFound = errors.New("course registration credentials not found")
var ErrCourseGrabItemNotFound = errors.New("course grab item not found")

type CourseRegistrationCredentialsRepo interface {
	GetByUserID(ctx context.Context, userID int64) (*toolmodel.CourseRegistrationCredentials, error)
	Upsert(ctx context.Context, credentials *toolmodel.CourseRegistrationCredentials) error
}

type CourseGrabRepo interface {
	RecordLoginAttempt(ctx context.Context, userID int64) error
	SetWorkerStatus(ctx context.Context, userID int64, status string) error
	TouchPoll(ctx context.Context, userID int64, nextPollAt time.Time) error
	ListRunnableUserIDs(ctx context.Context) ([]int64, error)
	UpsertItem(ctx context.Context, item *toolmodel.CourseGrabItem) error
	ListItems(ctx context.Context, userID int64) ([]toolmodel.CourseGrabItem, error)
	ListItemsByStatus(ctx context.Context, userID int64, status string) ([]toolmodel.CourseGrabItem, error)
	GetItem(ctx context.Context, userID, itemID int64) (*toolmodel.CourseGrabItem, error)
	UpdateItem(ctx context.Context, item *toolmodel.CourseGrabItem) error
	SetItemStatus(ctx context.Context, userID, itemID int64, status, message string) (*toolmodel.CourseGrabItem, error)
	GetJobSummary(ctx context.Context, userID int64) (toolmodel.CourseGrabJobSummary, error)
}
