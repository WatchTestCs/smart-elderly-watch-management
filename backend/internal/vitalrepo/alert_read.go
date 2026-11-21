package vitalrepo

import (
	"context"
	"database/sql"
	"errors"

	"suining-care/internal/db"
	"suining-care/internal/vitalpolicy"
)

type Alert = vitalpolicy.Alert

const vitalAlertSelect = `SELECT r.id, r.device_id, r.old_id, r.hr, r.bp, r.rate_state, r.pre_state,
		COALESCE((r.rate_handled_at IS NOT NULL AND r.create_time <= r.rate_handled_at), false),
		COALESCE((r.pre_handled_at IS NOT NULL AND r.create_time <= r.pre_handled_at), false),
		DATE_FORMAT(r.create_time,'%Y-%m-%d %H:%i:%s'), o.name, w.hrf, w.hrl, w.bpf, w.bpl
		FROM ratepre r LEFT JOIN oldperson o ON o.old_id=r.old_id
		LEFT JOIN watch_info w ON w.device_id=r.device_id`

func FindAlert(ctx context.Context, store db.RowQuerier, recordID int64) (Alert, bool, error) {
	return scanAlert(store.QueryRowContext(ctx, vitalAlertSelect+` WHERE r.id=?`, recordID))
}

func FindLatestPending(ctx context.Context, store db.RowQuerier, residentID int64, kind string) (Alert, bool, error) {
	condition := "(" + pendingRate + " OR " + pendingPressure + ")"
	switch kind {
	case "rate":
		condition = pendingRate
	case "pre":
		condition = pendingPressure
	}
	return scanAlert(store.QueryRowContext(ctx, vitalAlertSelect+
		` WHERE r.old_id=? AND `+condition+` ORDER BY r.create_time DESC, r.id DESC LIMIT 1`, residentID))
}

func scanAlert(row *sql.Row) (Alert, bool, error) {
	var item Alert
	err := row.Scan(&item.ID, &item.DeviceID, &item.OldID, &item.Hr, &item.Bp,
		&item.RateState, &item.PreState, &item.RateDone, &item.PreDone,
		&item.CreateTime, &item.OldPersonName, &item.HrHigh, &item.HrLow,
		&item.BpHigh, &item.BpLow)
	if errors.Is(err, sql.ErrNoRows) {
		return item, false, nil
	}
	return item, err == nil, err
}
