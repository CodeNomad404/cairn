package model

import "time"

const (
	CourseGrabStatusGrabbing  = "grabbing"
	CourseGrabStatusPaused    = "paused"
	CourseGrabStatusSucceeded = "succeeded"
	CourseGrabStatusNotFound  = "not_found"
)

type CourseGrabItem struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"-"`
	CourseName       string     `json:"course_name"`
	UpstreamCourseID string     `json:"upstream_course_id,omitempty"`
	Teacher          string     `json:"teacher"`
	Remaining        int        `json:"remaining"`
	Status           string     `json:"status"`
	Message          string     `json:"message"`
	AttemptCount     int        `json:"attempt_count"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`
	SucceededAt      *time.Time `json:"succeeded_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CourseGrabJobSummary struct {
	WorkerStatus string     `json:"worker_status"`
	LoginCount   int        `json:"login_count"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}
