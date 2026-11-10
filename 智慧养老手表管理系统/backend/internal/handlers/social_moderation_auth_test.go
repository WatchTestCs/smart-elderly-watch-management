package handlers

import (
	"net/http"
	"testing"

	"suining-care/internal/auth"
	"suining-care/internal/session"
)

func TestNoticeModerationRequiresLiveContentCapability(t *testing.T) {
	store := testDatabase(t)
	seedCareAdminDuty(t, store, 71, "supervisor")
	seedCareAdminDuty(t, store, 72, "caregiver")
	h := testHandler(t, store)

	tests := []struct {
		name string
		fn   http.HandlerFunc
		body map[string]any
	}{
		{"moderation_read", h.searchNoticeComments, map[string]any{"page": 1, "pageSize": 20}},
		{"comment_delete", h.deleteNoticeComment, map[string]any{"saidId": "1"}},
		{"comment_reply", h.replyToNoticeComment, map[string]any{"saidId": "1", "back": "reviewed"}},
		{"notice_delete", h.deleteNotice, map[string]any{"textId": "1"}},
	}
	sup := session.Principal{Role: auth.RoleAdmin, ID: 71}
	for _, tc := range tests {
		got := callAPI(withHealthPrincipal(h, sup, tc.fn), tc.body)
		if got.Code != http.StatusOK {
			t.Fatalf("supervisor %s HTTP %d: %s", tc.name, got.Code, got.Body.String())
		}
	}

	mustExec(t, store, "UPDATE care_staff_permission SET duty='auditor' WHERE admin_id=71")
	for _, actor := range []int64{71, 72} {
		p := session.Principal{Role: auth.RoleAdmin, ID: actor}
		for _, tc := range tests {
			got := callAPI(withHealthPrincipal(h, p, tc.fn), tc.body)
			if got.Code != http.StatusForbidden {
				t.Fatalf("actor=%d %s HTTP %d: %s", actor, tc.name, got.Code, got.Body.String())
			}
		}
	}
}
