package metric

import (
	"net/http"
	"slices"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	globalRegistry = initRegistry()
)

func initRegistry() *Registry {

	commonDims := []string{"repo_id", "plan_id"}

	registry := &Registry{
		reg: prometheus.NewRegistry(),
		backupBytesProcessed: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name: "backrest_backup_bytes_processed",
			Help: "The total number of bytes processed during a backup",
		}, commonDims),
		backupBytesAdded: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name: "backrest_backup_bytes_added",
			Help: "The total number of bytes added during a backup",
		}, commonDims),
		backupFileWarnings: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name: "backrest_backup_file_warnings",
			Help: "The total number of file warnings during a backup",
		}, commonDims),
		tasksDuration: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name: "backrest_tasks_duration_secs",
			Help: "The duration of a task in seconds",
		}, append(slices.Clone(commonDims), "task_type")),
		tasksRun: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "backrest_tasks_run_total",
			Help: "The total number of tasks run",
		}, append(slices.Clone(commonDims), "task_type", "status")),
	}

	registry.reg.MustRegister(registry.backupBytesProcessed)
	registry.reg.MustRegister(registry.backupBytesAdded)
	registry.reg.MustRegister(registry.backupFileWarnings)
	registry.reg.MustRegister(registry.tasksDuration)
	registry.reg.MustRegister(registry.tasksRun)

	return registry
}

func GetRegistry() *Registry {
	return globalRegistry
}

type Registry struct {
	reg                  *prometheus.Registry
	backupBytesProcessed *prometheus.SummaryVec
	backupBytesAdded     *prometheus.SummaryVec
	backupFileWarnings   *prometheus.SummaryVec
	tasksDuration        *prometheus.SummaryVec
	tasksRun             *prometheus.CounterVec
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

func (r *Registry) RecordTaskRun(repoID, planID, taskType string, duration_secs float64, status string) {
	if repoID == "" {
		repoID = "_unassociated_"
	}
	if planID == "" {
		planID = "_unassociated_"
	}
	r.tasksRun.WithLabelValues(repoID, planID, taskType, status).Inc()
	r.tasksDuration.WithLabelValues(repoID, planID, taskType).Observe(duration_secs)
}

func (r *Registry) RecordBackupSummary(repoID, planID string, bytesProcessed, bytesAdded int64, fileWarnings int64) {
	r.backupBytesProcessed.WithLabelValues(repoID, planID).Observe(float64(bytesProcessed))
	r.backupBytesAdded.WithLabelValues(repoID, planID).Observe(float64(bytesAdded))
	r.backupFileWarnings.WithLabelValues(repoID, planID).Observe(float64(fileWarnings))
}
