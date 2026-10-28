package handlers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"suining-care/internal/db"
)

// Health writes need six authoritative fields, not the UI profile, coordinates,
// counters or application metadata. Keep the same lookup keys, row-lock order,
// and credential policy as the full device repository.
type healthDevice struct {
	ID          int64
	No, Secret  string
	OldID       *int64
	Status      *int
	HealthEpoch int64
}

const healthDeviceCols = `device_id,device_no,secret,old_id,status,health_epoch`

func scanHealthDevice(row db.Row) (*healthDevice, error) {
	var d healthDevice
	if err := row.Scan(&d.ID, &d.No, &d.Secret, &d.OldID, &d.Status, &d.HealthEpoch); err != nil {
		return nil, err
	}
	return &d, nil
}
func (h *Handler) healthDeviceAuth(ctx context.Context, q oldQuerier, deviceNo, secret, disabledMsg string) (*healthDevice, string, string, error) {
	d, err := scanHealthDevice(q.QueryRow(ctx, `SELECT `+healthDeviceCols+` FROM device WHERE device_no=$1`, deviceNo))
	if err == pgx.ErrNoRows {
		d = nil
		err = nil
	}
	if err != nil {
		return nil, "", "", err
	}
	if d == nil {
		return nil, "0004", "设备号或密钥不正确", nil
	}
	code, msg := deviceCredentialFailure(secret, d.Secret, d.Status, disabledMsg)
	if code != "" {
		return nil, code, msg, nil
	}
	return d, "", "", nil
}

// Health requests never supply battery/appVersion. Avoid transmitting nil
// parameters and evaluating COALESCE for columns whose values stay untouched.
// Two equal timestamp arguments keep positional binding ordered on both engines.
func touchHealthDevice(ctx context.Context, q oldQuerier, deviceID int64, now time.Time) error {
	_, err := q.Exec(ctx, `UPDATE device SET last_time=$1,last_online_at=$2 WHERE device_id=$3`, now, now, deviceID)
	return err
}
