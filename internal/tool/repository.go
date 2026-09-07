package tool

import (
	"context"
	"errors"

	toolmodel "cairn.cn/internal/tool/model"
)

var ErrCourseRegistrationCredentialsNotFound = errors.New("course registration credentials not found")

type CourseRegistrationCredentialsRepo interface {
	GetByUserID(ctx context.Context, userID int64) (*toolmodel.CourseRegistrationCredentials, error)
	Upsert(ctx context.Context, credentials *toolmodel.CourseRegistrationCredentials) error
}
