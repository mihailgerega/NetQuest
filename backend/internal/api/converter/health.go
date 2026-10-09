package converter

import (
	healthv1 "github.com/netquest/netquest/backend/internal/contract/health/v1"
	"github.com/netquest/netquest/backend/internal/model"
)

// HealthReportToDTO переводит результат проверки зависимостей в DTO ответа.
func HealthReportToDTO(report model.HealthReport) healthv1.Report {
	checks := make(map[string]healthv1.ComponentCheck, len(report.Checks))
	for name, check := range report.Checks {
		checks[name] = healthv1.ComponentCheck{
			Status:    check.Status,
			LatencyMs: check.LatencyMs,
			Error:     check.Error,
		}
	}

	return healthv1.Report{
		Status:    report.Status,
		Checks:    checks,
		Timestamp: report.Timestamp,
	}
}
