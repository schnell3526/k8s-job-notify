package notifier

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
)

// Notifier is the interface for sending job notifications.
// Implement this interface to add new notification destinations
// (e.g., Discord, Microsoft Teams, PagerDuty, etc.)
type Notifier interface {
	// NotifyJobCompleted sends a notification when a job completes.
	// succeeded is true if the job completed successfully, false if it failed.
	NotifyJobCompleted(ctx context.Context, job *batchv1.Job, succeeded bool) error
}
