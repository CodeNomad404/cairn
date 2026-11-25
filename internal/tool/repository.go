package tool

import (
	"context"
	"errors"
	"time"

	toolmodel "cairn.cn/internal/tool/model"
)

var ErrCourseRegistrationCredentialsNotFound = errors.New("course registration credentials not found")
var ErrCourseGrabItemNotFound = errors.New("course grab item not found")
var ErrAccountingBookNotFound = errors.New("accounting book not found")
var ErrAccountingExpenseNotFound = errors.New("accounting expense not found")

type CourseRegistrationCredentialsRepo interface {
	GetByUserID(ctx context.Context, userID int64) (*toolmodel.CourseRegistrationCredentials, error)
	Upsert(ctx context.Context, credentials *toolmodel.CourseRegistrationCredentials) error
	BindMIS(ctx context.Context, credentials *toolmodel.CourseRegistrationCredentials) error
}

var ErrHomeworkNotFound = errors.New("homework not found")
var ErrMISBindingChanged = errors.New("MIS binding changed during synchronization")

type HomeworkRepo interface {
	Snapshot(ctx context.Context, userID int64) (toolmodel.HomeworkSnapshot, error)
	Get(ctx context.Context, userID, id int64) (*toolmodel.Homework, error)
	Prepare(ctx context.Context, userID, version int64, term string, courses []toolmodel.HomeworkCourse) error
	SaveCourse(ctx context.Context, userID, version int64, term string, course toolmodel.HomeworkCourse, items []toolmodel.Homework, syncError string) error
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

type ClassroomAvailabilityQuery struct {
	Weekday     int
	Period      int
	BuildingID  string
	MinCapacity int
}

type ClassroomAvailabilityRepo interface {
	ReplaceSnapshot(ctx context.Context, items []toolmodel.ClassroomAvailability) error
	List(ctx context.Context, query ClassroomAvailabilityQuery) ([]toolmodel.ClassroomAvailability, *time.Time, error)
	LastSyncedAt(ctx context.Context) (*time.Time, error)
}

type AccountingRepo interface {
	CreateBook(ctx context.Context, book *toolmodel.AccountingBook) error
	GetBook(ctx context.Context, userID, bookID int64) (*toolmodel.AccountingBookSummary, error)
	ListBooks(ctx context.Context, userID int64) ([]toolmodel.AccountingBookSummary, error)
	UpdateBook(ctx context.Context, book *toolmodel.AccountingBook) error
	DeleteBook(ctx context.Context, userID, bookID int64) error
	ListBookCategoryStatistics(ctx context.Context, userID, bookID int64) ([]toolmodel.AccountingCategoryStatistic, error)

	CreateExpense(ctx context.Context, userID int64, expense *toolmodel.AccountingExpense) error
	ListExpenses(ctx context.Context, userID, bookID int64, category toolmodel.AccountingExpenseCategory, limit, offset int) ([]toolmodel.AccountingExpense, error)
	UpdateExpense(ctx context.Context, userID int64, expense *toolmodel.AccountingExpense) error
	DeleteExpense(ctx context.Context, userID, expenseID int64) error

	GetMonthlyStatistics(ctx context.Context, userID int64, start, end time.Time) (toolmodel.AccountingMonthlyStatistics, error)
}

const (
	TransferFileLimit     int64 = 100 << 20
	TransferTextLimit           = 100 << 10
	TransferRetentionDays       = 15
)

var (
	ErrTransferNotFound  = errors.New("transfer resource not found")
	ErrTransferDuplicate = errors.New("transfer channel name already exists")
)

type TransferRepo interface {
	CreateChannel(ctx context.Context, userID int64, name string) (*toolmodel.TransferChannel, error)
	ListChannels(ctx context.Context, userID int64) ([]toolmodel.TransferChannel, error)
	RenameChannel(ctx context.Context, userID, id int64, name string) (*toolmodel.TransferChannel, error)
	DeleteChannel(ctx context.Context, userID, id int64) error
	CreateItem(ctx context.Context, userID int64, item *toolmodel.TransferItem) error
	Activate(ctx context.Context, id, size int64) error
	Abandon(ctx context.Context, id int64) error
	ListItems(ctx context.Context, userID, channelID int64, kind string, page, pageSize int) ([]toolmodel.TransferItem, error)
	GetItem(ctx context.Context, userID, id int64) (*toolmodel.TransferItem, error)
	DeleteItem(ctx context.Context, userID, id int64) error
	Usage(ctx context.Context, userID int64) (int64, error)
	CleanupItems(ctx context.Context) ([]toolmodel.TransferItem, error)
	Purge(ctx context.Context, id int64) error
}
