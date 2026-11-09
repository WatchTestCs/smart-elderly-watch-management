package vitalrepo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"suining-care/internal/db"
)

type Alert struct {
	ID            int64
	DeviceID      *int64
	OldID         *int64
	Hr            *int64
	Bp            *int64
	RateState     *string
	PreState      *string
	RateDone      bool
	PreDone       bool
	CreateTime    *string
	OldPersonName *string
	HrHigh, HrLow *int
	BpHigh, BpLow *int
}

const vitalAlertSelect = `SELECT r.id, r.device_id, r.old_id, r.hr, r.bp, r.rate_state, r.pre_state,
		COALESCE((r.rate_handled_at IS NOT NULL AND r.create_time <= r.rate_handled_at), false),
		COALESCE((r.pre_handled_at IS NOT NULL AND r.create_time <= r.pre_handled_at), false),
		TO_CHAR(r.create_time,'YYYY-MM-DD HH24:MI:SS'), o.name, w.hrf, w.hrl, w.bpf, w.bpl
		FROM ratepre r LEFT JOIN oldperson o ON o.old_id=r.old_id
		LEFT JOIN watch_info w ON w.device_id=r.device_id`

func FindAlert(ctx context.Context, store *db.Database, recordID int64) (Alert, bool, error) {
	return scanAlert(store.QueryRow(ctx, vitalAlertSelect+` WHERE r.id=$1`, recordID))
}

func FindLatestPending(ctx context.Context, store *db.Database, residentID int64, kind string) (Alert, bool, error) {
	condition := "(" + pendingRate + " OR " + pendingPressure + ")"
	switch kind {
	case "rate":
		condition = pendingRate
	case "pre":
		condition = pendingPressure
	}
	return scanAlert(store.QueryRow(ctx, vitalAlertSelect+
		` WHERE r.old_id=$1 AND `+condition+` ORDER BY r.create_time DESC, r.id DESC LIMIT 1`, residentID))
}

func scanAlert(row db.Row) (Alert, bool, error) {
	var item Alert
	err := row.Scan(&item.ID, &item.DeviceID, &item.OldID, &item.Hr, &item.Bp,
		&item.RateState, &item.PreState, &item.RateDone, &item.PreDone,
		&item.CreateTime, &item.OldPersonName, &item.HrHigh, &item.HrLow,
		&item.BpHigh, &item.BpLow)
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return item, false, nil
	}
	return item, err == nil, err
}
