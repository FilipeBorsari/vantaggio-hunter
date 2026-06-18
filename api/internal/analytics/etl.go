package analytics

import (
	"context"
	"log/slog"
	"time"
)

// StartETLJob runs the ETL once immediately, then on a 1-hour ticker.
// It logs errors without stopping the loop so the server stays healthy.
func StartETLJob(ctx context.Context, svc ServiceInterface) {
	runOnce := func() {
		start := time.Now()
		if err := svc.RunETL(ctx); err != nil {
			slog.Error("etl run failed", "error", err, "elapsed", time.Since(start).String())
			return
		}
		slog.Info("etl run completed", "elapsed", time.Since(start).String())
	}

	// Run once at startup so the dashboard has data immediately.
	runOnce()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
