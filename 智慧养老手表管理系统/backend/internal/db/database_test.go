package db

import (
	"reflect"
	"strings"
	"testing"
)

func TestMySQLBoundParameters(t *testing.T) {
	q, args, err := mysqlQuery("SELECT '$1', $2, $1, $2, 'escaped''$3'", []any{"first", 7})
	if err != nil || q != "SELECT '$1', ?, ?, ?, 'escaped''$3'" || !reflect.DeepEqual(args, []any{7, "first", 7}) {
		t.Fatalf("query=%s args=%v err=%v", q, args, err)
	}
	if _, _, err = mysqlQuery("SELECT $2", []any{1}); err == nil {
		t.Fatal("missing parameter accepted")
	}
}

func TestMySQLDialect(t *testing.T) {
	for _, test := range []struct{ from, to string }{
		{"SELECT bool_or(status='sent') FROM care_notification_outbox", "SELECT max(status='sent') FROM care_notification_outbox"},
		{"SELECT id FROM care_incident FOR UPDATE SKIP LOCKED LIMIT 20", "SELECT id FROM care_incident LIMIT 20 FOR UPDATE SKIP LOCKED"},
		{"SELECT now()-interval '2 minutes'", "SELECT CURRENT_TIMESTAMP(6)-INTERVAL 2 MINUTE"},
		{"INSERT INTO watch_info(device_id) VALUES(1) ON CONFLICT(device_id) DO NOTHING", "INSERT INTO watch_info(device_id) VALUES(1) ON DUPLICATE KEY UPDATE id=id"},
		{"UPDATE care_notification_outbox SET retry_at=now()+($3 * interval '1 second')", "UPDATE care_notification_outbox SET retry_at=TIMESTAMPADD(SECOND,?,CURRENT_TIMESTAMP(6))"},
	} {
		q, _, err := mysqlQuery(test.from, []any{0, 0, 30})
		if err != nil || strings.TrimSpace(q) != test.to {
			t.Fatalf("%s => %s (%v)", test.from, q, err)
		}
	}
}
