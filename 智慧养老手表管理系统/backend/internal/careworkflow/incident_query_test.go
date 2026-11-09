package careworkflow

import (
	"strings"
	"testing"
)

func TestNotificationAggregateUsesNativeMySQL(t *testing.T) {
	if strings.Contains(notificationStatusSQL, "bool_or(") || !strings.Contains(notificationStatusSQL, "MAX(n.status='delivered')") {
		t.Fatalf("care notification aggregate is not native MySQL: %s", notificationStatusSQL)
	}
}

func TestActiveIncidentOrderPrioritizesRiskAndOverdueWork(t *testing.T) {
	for _, required := range []string{"i.kind IN ('sos','fall')", "i.severity='critical'", "i.severity IN ('high','warning')", "i.due_at < CURRENT_TIMESTAMP(6)", "i.due_at ASC"} {
		if !strings.Contains(activeIncidentOrderSQL, required) {
			t.Fatalf("active task priority ordering is missing %q: %s", required, activeIncidentOrderSQL)
		}
	}
}

func TestVerifiedIncidentOrderShowsNewestReviewFirst(t *testing.T) {
	if !strings.Contains(verifiedIncidentOrderSQL, "i.verified_at DESC") {
		t.Fatalf("verified history is not newest-first: %s", verifiedIncidentOrderSQL)
	}
}
