package careworkflow

import (
	"strings"
	"testing"
)

func TestNotificationAggregateUsesNativeSQL(t *testing.T) {
	mysql := notificationStatusSQL(true)
	postgres := notificationStatusSQL(false)
	if strings.Contains(mysql, "bool_or(") || !strings.Contains(mysql, "MAX(n.status='delivered')") {
		t.Fatalf("MySQL care notification aggregate is not native: %s", mysql)
	}
	if strings.Contains(postgres, "MAX(") || !strings.Contains(postgres, "bool_or(n.status='delivered')") {
		t.Fatalf("PostgreSQL care notification aggregate is not native: %s", postgres)
	}
}
