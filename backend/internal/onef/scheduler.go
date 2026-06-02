package onef

import (
	"context"
	"log/slog"
	"time"
)

// StartScheduler kicks off a goroutine that polls 1F every `interval`.
// The first tick fires after `interval` (NOT immediately) to keep server
// startup quiet; admins can force an immediate run via the manual endpoint.
// Stops cleanly when ctx is cancelled.
func StartScheduler(ctx context.Context, svc *Service, interval time.Duration, log *slog.Logger) {
	if !svc.Configured() {
		log.Info("1F scheduler disabled (no ONEF_BASE_URL)")
		return
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	go func() {
		log.Info("1F scheduler started", slog.Duration("interval", interval))
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("1F scheduler stopping")
				return
			case <-ticker.C:
				runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				res, err := svc.RunSync(runCtx, TriggerCron, nil)
				cancel()
				if err != nil {
					log.Error("1F cron sync failed", slog.String("err", err.Error()))
					continue
				}
				log.Info("1F cron sync ok",
					slog.Int("fetched", res.FetchedCount),
					slog.Int("created", res.CreatedCount),
					slog.Int("updated", res.UpdatedCount),
					slog.Int("skipped", res.SkippedCount),
					slog.Int("duration_ms", res.DurationMS))
			}
		}
	}()
}
