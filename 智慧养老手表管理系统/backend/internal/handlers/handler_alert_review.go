package handlers

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"careold-go/internal/response"
	"github.com/jackc/pgx/v5"
)

func reviewVerdict(value string) bool { return value == "confirmed" || value == "false_alarm" }

// SOS and fence conclusions are recorded by the existing independent care verify
// transition. This endpoint only reviews a concrete legacy vital-sign record.
func (h *Handler) alertReview(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.careSupervisor(w, r)
	if !ok {
		return
	}
	var req struct {
		RecordID         int64  `json:"recordId"`
		Kind             string `json:"kind"`
		Verdict          string `json:"verdict"`
		Reason           string `json:"reason"`
		ExpectedReviewID int64  `json:"expectedReviewId"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.RecordID < 1 || (req.Kind != "rate" && req.Kind != "pre") || !reviewVerdict(req.Verdict) || req.Reason == "" || utf8.RuneCountInString(req.Reason) > 2000 || req.ExpectedReviewID < 0 {
		write400(w, "须指定具体心率/血压告警、结论、理由和上次复核 ID")
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	var record int64
	var state *string
	// Kind is allowlisted above, never arbitrary SQL input.
	err = tx.QueryRow(ctx, "SELECT id,"+req.Kind+"_state FROM ratepre WHERE id=$1 FOR UPDATE", req.RecordID).Scan(&record, &state)
	if err == pgx.ErrNoRows {
		response.JSON(w, 404, map[string]any{"message": "记录不存在"})
		return
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	var previous int64
	err = tx.QueryRow(ctx, "SELECT COALESCE(MAX(id),0) FROM alert_review WHERE entity='ratepre' AND record_id=$1 AND kind=$2", req.RecordID, req.Kind).Scan(&previous)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if previous != req.ExpectedReviewID {
		conflict409(w, "复核结论已变化，请刷新")
		return
	}
	if previous == 0 && (state == nil || *state != "1") {
		conflict409(w, "此记录没有可复核的活动告警")
		return
	}
	var id int64
	req.Reason, err = h.sealNote(req.Reason)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	err = tx.QueryRow(ctx, "INSERT INTO alert_review(entity,record_id,kind,verdict,reason,actor_id) VALUES('ratepre',$1,$2,$3,$4,$5) RETURNING id", req.RecordID, req.Kind, req.Verdict, req.Reason, actor).Scan(&id)
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, map[string]any{"reviewId": id, "verdict": req.Verdict, "alarmStateChanged": false})
}

func (h *Handler) alertReviews(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.careSupervisor(w, r); !ok {
		return
	}
	var req struct {
		Entity   string `json:"entity"`
		RecordID int64  `json:"recordId"`
		Kind     string `json:"kind"`
		After    int64  `json:"after"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if (req.Entity != "care" && req.Entity != "ratepre") || req.RecordID < 1 || req.After < 0 || len(req.Kind) > 32 {
		write400(w, "复核目标或游标无效")
		return
	}
	rows, err := h.DB.Query(r.Context(), "SELECT id,kind,verdict,reason,actor_id,created_at FROM alert_review WHERE entity=$1 AND record_id=$2 AND ($3='' OR kind=$3) AND id>$4 ORDER BY id LIMIT 100", req.Entity, req.RecordID, req.Kind, req.After)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, actor int64
		var kind, verdict, reason string
		var at time.Time
		if rows.Scan(&id, &kind, &verdict, &reason, &actor, &at) != nil {
			response.Error500(w, ref())
			return
		}
		reason, err = h.openNote(reason)
		if err != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{"id": id, "kind": kind, "verdict": verdict, "reason": reason, "actorId": actor, "createdAt": at})
	}
	if !rowsOK(w, rows) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
