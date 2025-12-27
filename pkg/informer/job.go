package informer

import (
	"context"
	"log/slog"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/schnell3526/k8s-job-notify/pkg/config"
	"github.com/schnell3526/k8s-job-notify/pkg/notifier"
)

type JobInformer struct {
	clientset         kubernetes.Interface
	notifier          notifier.Notifier
	resyncPeriod      time.Duration
	namespace         string
	notificationLevel config.NotificationLevel

	notifiedJobs map[string]struct{}
	mu           sync.RWMutex
}

func NewJobInformer(
	clientset kubernetes.Interface,
	notifier notifier.Notifier,
	namespace string,
	resyncPeriod time.Duration,
	notificationLevel config.NotificationLevel,
) *JobInformer {
	return &JobInformer{
		clientset:         clientset,
		notifier:          notifier,
		namespace:         namespace,
		resyncPeriod:      resyncPeriod,
		notificationLevel: notificationLevel,
		notifiedJobs:      make(map[string]struct{}),
	}
}

func (j *JobInformer) Run(ctx context.Context) error {
	var factory informers.SharedInformerFactory
	if j.namespace == "" {
		factory = informers.NewSharedInformerFactory(j.clientset, j.resyncPeriod)
	} else {
		factory = informers.NewSharedInformerFactoryWithOptions(
			j.clientset,
			j.resyncPeriod,
			informers.WithNamespace(j.namespace),
		)
	}

	jobInformer := factory.Batch().V1().Jobs().Informer()

	_, err := jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: j.handleJobUpdate,
	})
	if err != nil {
		return err
	}

	slog.Info("starting job informer",
		"namespace", j.namespaceLogValue(),
		"resync_period", j.resyncPeriod,
	)

	factory.Start(ctx.Done())
	factory.WaitForCacheSync(ctx.Done())

	<-ctx.Done()
	slog.Info("job informer stopped")
	return nil
}

func (j *JobInformer) handleJobUpdate(oldObj, newObj interface{}) {
	oldJob, ok := oldObj.(*batchv1.Job)
	if !ok {
		return
	}
	newJob, ok := newObj.(*batchv1.Job)
	if !ok {
		return
	}

	jobKey := jobKeyFunc(newJob)

	if j.isAlreadyNotified(jobKey) {
		return
	}

	succeeded := j.isJobSucceeded(oldJob, newJob)
	failed := j.isJobFailed(oldJob, newJob)

	if !succeeded && !failed {
		return
	}

	j.markAsNotified(jobKey)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if succeeded && j.notificationLevel.ShouldNotifySuccess() {
		slog.Info("job succeeded, sending notification",
			"job", newJob.Name,
			"namespace", newJob.Namespace,
		)
		if err := j.notifier.NotifyJobCompleted(ctx, newJob, true); err != nil {
			slog.Error("failed to send success notification",
				"job", newJob.Name,
				"namespace", newJob.Namespace,
				"error", err,
			)
		}
	} else if failed && j.notificationLevel.ShouldNotifyFailure() {
		slog.Info("job failed, sending notification",
			"job", newJob.Name,
			"namespace", newJob.Namespace,
		)
		if err := j.notifier.NotifyJobCompleted(ctx, newJob, false); err != nil {
			slog.Error("failed to send failure notification",
				"job", newJob.Name,
				"namespace", newJob.Namespace,
				"error", err,
			)
		}
	}
}

func (j *JobInformer) isJobSucceeded(oldJob, newJob *batchv1.Job) bool {
	if oldJob.Status.Succeeded > 0 {
		return false
	}
	return newJob.Status.Succeeded > 0
}

func (j *JobInformer) isJobFailed(oldJob, newJob *batchv1.Job) bool {
	if isJobConditionFailed(oldJob) {
		return false
	}
	return isJobConditionFailed(newJob)
}

func isJobConditionFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == "True" {
			return true
		}
	}
	return false
}

func (j *JobInformer) isAlreadyNotified(key string) bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	_, exists := j.notifiedJobs[key]
	return exists
}

func (j *JobInformer) markAsNotified(key string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.notifiedJobs[key] = struct{}{}
}

func (j *JobInformer) namespaceLogValue() string {
	if j.namespace == "" {
		return "all"
	}
	return j.namespace
}

func jobKeyFunc(job *batchv1.Job) string {
	return job.Namespace + "/" + job.Name
}

// Ensure metav1 is used (for future expansion)
var _ = metav1.Now
