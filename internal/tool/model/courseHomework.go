package model

import "time"

type Homework struct {
	ID             int64
	UserID         int64
	BindingVersion int64
	TermCode       string
	CourseID       string
	CourseName     string
	PlatformID     int64
	TeacherID      int64
	Title          string
	Content        string
	OpensAt        *time.Time
	Deadline       *time.Time
	SubmittedAt    *time.Time
	Submission     string
	Grading        string
}

type HomeworkCourse struct {
	ID            string
	Name          string
	CourseNumber  string
	GroupID       string
	LastAttemptAt time.Time
	LastSuccessAt *time.Time
	Error         string
}

type HomeworkSnapshot struct {
	TermCode      string
	LastSuccessAt *time.Time
	HasSnapshot   bool
	Courses       []HomeworkCourse
	Items         []Homework
}
