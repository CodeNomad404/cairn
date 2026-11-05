package logic

import (
	"context"
	"strings"

	"cairn.cn/internal/httpx"
	toolmodel "cairn.cn/internal/tool/model"
)

type MISBindResult struct {
	Bound     bool   `json:"bound"`
	StudentID string `json:"student_id"`
}

func (l *CourseRegistrationLogic) BindMIS(ctx context.Context, userID int64, studentID, password string) (any, httpx.Code, error) {
	credentials := &toolmodel.CourseRegistrationCredentials{UserID: userID, StudentID: strings.TrimSpace(studentID), BJTUPassword: password}
	client, code, err := l.authenticateMIS(ctx, credentials)
	if err != nil {
		return nil, code, err
	}
	if err := l.credentialsRepo.BindMIS(ctx, credentials); err != nil {
		return nil, httpx.CodeInternalError, httpx.NewAppError("保存 MIS 绑定失败", err)
	}
	l.mu.Lock()
	l.sessions[userID] = client
	l.mu.Unlock()
	return MISBindResult{Bound: true, StudentID: maskedMISStudentID(credentials.StudentID)}, httpx.CodeSuccess, nil
}

func maskedMISStudentID(studentID string) string {
	id := []rune(studentID)
	masked := "****"
	if len(id) > 4 {
		masked = string(id[:2]) + "****" + string(id[len(id)-2:])
	}
	return masked
}
