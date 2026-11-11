package careworkflow

import (
	"strings"
	"testing"

	"suining-care/internal/carepolicy"
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

func TestCaregiverScopeUsesIndexedVisibilityBucket(t *testing.T) {
	scope, args := incidentReadScope(carepolicy.Caregiver, 42)
	if strings.Contains(scope, " OR ") || !strings.Contains(scope, "visibility_group IN") {
		t.Fatalf("caregiver scope did not use the indexed visibility bucket: %s", scope)
	}
	if len(args) != 1 || args[0] != "ASSIGNED:42" {
		t.Fatalf("unexpected caregiver visibility bucket args: %#v", args)
	}
}
