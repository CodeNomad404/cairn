package app

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	authhandler "cairn.cn/internal/auth/handler"
	authlogic "cairn.cn/internal/auth/logic"
	authrepo "cairn.cn/internal/auth/repository"
	bloghandler "cairn.cn/internal/blog/handler"
	bloglogic "cairn.cn/internal/blog/logic"
	blogrepo "cairn.cn/internal/blog/repository"
	"cairn.cn/internal/config"
	"cairn.cn/internal/seo"
	toolhandler "cairn.cn/internal/tool/handler"
	toollogic "cairn.cn/internal/tool/logic"
	toolrepo "cairn.cn/internal/tool/repository"
)

type applicationComponents struct {
	accountingHandler       *toolhandler.AccountingHandler
	authLogic               *authlogic.AuthLogic
	authHandler             *authhandler.AuthHandler
	postHandler             *bloghandler.PostHandler
	commentHandler          *bloghandler.CommentHandler
	toolHandler             *toolhandler.Handler
	courseHandler           *toolhandler.CourseHandler
	classroomHandler        *toolhandler.ClassroomHandler
	seoHandler              *seo.Handler
	courseRegistrationLogic *toollogic.CourseRegistrationLogic
	classroomLogic          *toollogic.ClassroomLogic
}

// newApplicationComponents 创建应用程序组件。
func newApplicationComponents(
	cfg *config.Config,
	db *pgxpool.Pool,
	logger *slog.Logger,
) *applicationComponents {
	userRepo := authrepo.NewUserRepo(db)
	sessionRepo := authrepo.NewSessionRepo(db)
	postRepo := blogrepo.New(db)
	commentRepo := blogrepo.NewCommentRepo(db)
	courseRegistrationCredentialsRepo := toolrepo.NewCourseRegistrationCredentialsRepo(db)
	courseGrabRepo := toolrepo.NewCourseGrabRepo(db)
	classroomAvailabilityRepo := toolrepo.NewClassroomAvailabilityRepo(db)
	accountingRepo := toolrepo.NewAccountingRepo(db)

	accountingLogic := toollogic.NewAccountingLogic(accountingRepo)
	authLogic := authlogic.NewAuthLogic(userRepo, sessionRepo)
	postLogic := bloglogic.NewPostLogic(postRepo)
	commentLogic := bloglogic.NewCommentLogic(postRepo, commentRepo)
	jsonLogic := toollogic.NewJSONLogic()
	courseRegistrationLogic := toollogic.NewCourseRegistrationLogic(
		courseRegistrationCredentialsRepo, courseGrabRepo, logger,
	)
	classroomLogic := toollogic.NewClassroomLogic(
		classroomAvailabilityRepo, courseRegistrationLogic, logger,
	)

	return &applicationComponents{
		accountingHandler:       toolhandler.NewAccountingHandler(accountingLogic),
		authLogic:               authLogic,
		authHandler:             authhandler.NewAuthHandler(authLogic, cfg.IsProduction()),
		postHandler:             bloghandler.NewPostHandler(postLogic),
		commentHandler:          bloghandler.NewCommentHandler(commentLogic),
		toolHandler:             toolhandler.New(jsonLogic),
		courseHandler:           toolhandler.NewCourseHandler(courseRegistrationLogic),
		classroomHandler:        toolhandler.NewClassroomHandler(classroomLogic),
		seoHandler:              seo.New(postLogic),
		courseRegistrationLogic: courseRegistrationLogic,
		classroomLogic:          classroomLogic,
	}
}
