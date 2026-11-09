package fencerepo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/db"
)

var ErrFenceLimit = errors.New("too many fences for binding")
var ErrFenceRevision = errors.New("fence or binding revision conflict")

// Save runs under a previously acquired device row lock and remains in the
// same transaction as the caller's care-access audit.
func Save(ctx context.Context, tx *db.Transaction, actor int64, deviceID, residentID, epoch int64, config *Config) error {
	coordinates, err := json.Marshal(config.Coordinates)
	if err != nil {
		return err
	}
	if config.ID == 0 {
		var count int
		if err := tx.QueryRow(ctx,
			"SELECT count(*) FROM device_fence WHERE device_id=$1 AND epoch=$2",
			deviceID, epoch).Scan(&count); err != nil {
			return err
		}
		if count >= 20 {
			return ErrFenceLimit
		}
		err := tx.QueryRow(ctx,
			`INSERT INTO device_fence(device_id,old_id,epoch,name,enabled,coordinates,coordinate_system,timezone,time_begin,time_end,min_points,dwell_seconds,max_accuracy,updated_by)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`,
			deviceID, residentID, epoch, config.Name, config.Enabled, json.RawMessage(coordinates),
			config.CoordinateSystem, config.Timezone, config.TimeBegin, config.TimeEnd, config.MinPoints,
			config.DwellSeconds, config.MaxAccuracy, actor).Scan(&config.ID)
		if err != nil {
			return err
		}
		config.Revision = 1
		return nil
	}
	var revision int64
	err = tx.QueryRow(ctx,
		"SELECT revision FROM device_fence WHERE id=$1 AND device_id=$2 AND old_id=$3 AND epoch=$4 FOR UPDATE",
		config.ID, deviceID, residentID, epoch).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && revision != config.Revision) {
		return ErrFenceRevision
	}
	if err != nil {
		return err
	}
	config.Revision++
	if _, err := tx.Exec(ctx,
		`UPDATE device_fence SET name=$2,enabled=$3,coordinates=$4,coordinate_system=$5,timezone=$6,time_begin=$7,time_end=$8,min_points=$9,dwell_seconds=$10,max_accuracy=$11,updated_by=$12,revision=$13,updated_at=now() WHERE id=$1`,
		config.ID, config.Name, config.Enabled, json.RawMessage(coordinates), config.CoordinateSystem,
		config.Timezone, config.TimeBegin, config.TimeEnd, config.MinPoints, config.DwellSeconds,
		config.MaxAccuracy, actor, config.Revision); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "DELETE FROM fence_state WHERE fence_id=$1", config.ID)
	return err
}
