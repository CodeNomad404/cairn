package app

import (
	"context"
	"fmt"

	"cairn.cn/internal/config"
)

func startBackgroundWorkers(
	ctx context.Context,
	cfg *config.Config,
	components *applicationComponents,
) error {
	if err := components.courseRegistrationLogic.StartWorkers(ctx); err != nil {
		return fmt.Errorf("start course grab workers: %w", err)
	}
	if err := components.transferLogic.StartCleanup(ctx); err != nil {
		return fmt.Errorf("start transfer cleanup: %w", err)
	}
	components.classroomLogic.StartScheduler(ctx, cfg.ClassroomSync.UserID)
	return nil
}
