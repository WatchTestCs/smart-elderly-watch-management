package fencerepo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"suining-care/internal/db"
	"suining-care/internal/fence"
)

// LoadState locks the persisted state so observations for a fence serialize.
// A different revision starts with a fresh state after configuration changes.
func LoadState(ctx context.Context, tx *db.Transaction, fenceID, revision int64) (fence.State, error) {
	var storedRevision int64
	var payload []byte
	err := tx.QueryRow(ctx, "SELECT revision,payload FROM fence_state WHERE fence_id=$1 FOR UPDATE", fenceID).Scan(&storedRevision, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return fence.State{}, nil
	}
	if err != nil {
		return fence.State{}, err
	}
	if storedRevision != revision {
		return fence.State{}, nil
	}
	var state fence.State
	if err := json.Unmarshal(payload, &state); err != nil {
		return fence.State{}, err
	}
	return state, nil
}

// SaveState writes the observation in the transaction that owns the fence lock.
func SaveState(ctx context.Context, tx *db.Transaction, fenceID, revision int64, state fence.State) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO fence_state(fence_id,revision,payload) VALUES($1,$2,$3)
		 ON CONFLICT(fence_id) DO UPDATE SET revision=EXCLUDED.revision,payload=EXCLUDED.payload`,
		fenceID, revision, json.RawMessage(payload))
	return err
}
