package handlers

import (
	"context"
	"net/http"
	"testing"
)

func TestReportPayloadDigestIsCanonical(t *testing.T) {
	first := map[string]any{"deviceNo": "WATCH001", "hr": 70, "step": 12}
	reordered := map[string]any{"step": 12, "deviceNo": "WATCH001", "hr": 70}
	a, err := reportPayloadDigest(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := reportPayloadDigest(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if a != b || len(a) != 64 {
		t.Fatalf("digest not canonical: %q vs %q", a, b)
	}
	altered := map[string]any{"deviceNo": "WATCH001", "hr": 71, "step": 12}
	c, err := reportPayloadDigest(altered)
	if err != nil {
		t.Fatal(err)
	}
	if c == a {
		t.Fatal("changed payload produced the same digest")
	}
}

func numberedReportPayload(requestID string) map[string]any {
	return map[string]any{
		"deviceNo": "WATCH001", "secret": testSecret,
		"bindingVersion": 1, "hr": 74, "step": 12,
		"requestId": requestID,
	}
}

// Exercise the full receipt lifecycle on one isolated database, including
// an independent handler instance and the legacy unnumbered path.
func TestReportReceiptLifecycle(t *testing.T) {
	pool := testDatabase(t)
	seedDevice(t, pool)
	h := testHandler(t, pool)
	ctx := context.Background()
	assertCounts := func(samples, receipts int) {
		t.Helper()
		for _, check := range []struct {
			query string
			want  int
		}{
			{"SELECT count(*) FROM ratepre", samples},
			{"SELECT count(*) FROM device_report_receipt", receipts},
		} {
			var count int
			if err := pool.QueryRowContext(ctx, check.query).Scan(&count); err != nil || count != check.want {
				t.Fatalf("%s: count=%d want=%d err=%v", check.query, count, check.want, err)
			}
		}
	}
	payload := numberedReportPayload("receipt-lifecycle")
	first := api(t, h.deviceReport, payload)
	if first["duplicate"] == true {
		t.Fatalf("first request marked duplicate: %v", first)
	}
	assertCounts(1, 1)

	// An independent instance must replay the persisted acknowledgement,
	// rather than relying on the original process's memory.
	replica := testHandler(t, pool)
	second := api(t, replica.deviceReport, payload)
	if second["duplicate"] != true {
		t.Fatalf("replay did not return the committed acknowledgement: %v", second)
	}
	assertCounts(1, 1)

	altered := numberedReportPayload("receipt-lifecycle")
	altered["hr"] = 140
	rec := callAPI(replica.deviceReport, altered)
	if rec.Code != http.StatusConflict {
		t.Fatalf("ID reuse with different payload accepted: %d %s", rec.Code, rec.Body.String())
	}
	assertCounts(1, 1)
	if retry := api(t, h.deviceReport, payload); retry["duplicate"] != true {
		t.Fatalf("rejected reuse destroyed the original receipt: %v", retry)
	}
	assertCounts(1, 1)

	delete(payload, "requestId")
	first = api(t, h.deviceReport, payload)
	second = api(t, replica.deviceReport, payload)
	if first["duplicate"] == true || second["duplicate"] == true {
		t.Fatalf("unnumbered report unexpectedly deduplicated: %v / %v", first, second)
	}
	assertCounts(3, 1)
}
