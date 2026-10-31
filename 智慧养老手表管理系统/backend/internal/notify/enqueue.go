package notify

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strings"
	"suining-care/internal/db"
	"suining-care/internal/securestore"
)

func EnqueueChannels(ctx context.Context, tx *db.Transaction, box *securestore.Box, topic, key string, familyID int64, phone string, channels []string) error {
	return EnqueueBatch(ctx, tx, box, []Message{{Topic: topic, Key: key, FamilyID: familyID, Phone: phone, Channels: channels}})
}

// Message is one recipient's durable notification. A nil Channels uses sms and in_app.
type Message struct {
	Topic, Key string
	FamilyID   int64
	Phone      string
	Channels   []string
}

// EnqueueBatch shares the caller's transaction; success still requires its commit.
// Statements are bounded, but no messages are discarded when the input is larger.
func EnqueueBatch(ctx context.Context, tx *db.Transaction, box *securestore.Box, messages []Message) error {
	if !tx.IsMySQL() {
		return nil
	}
	var values strings.Builder
	args := make([]any, 0, min(len(messages), 256)*10)
	flush := func() error {
		if len(args) == 0 {
			return nil
		}
		_, err := tx.Exec(ctx, `INSERT INTO message_outbox(topic,source_key,family_id,channel,recipient,status,delivered_at) VALUES `+values.String()+` ON CONFLICT(topic,source_key,family_id,channel) DO NOTHING`, args...)
		values.Reset()
		args = args[:0]
		return err
	}
	for _, message := range messages {
		recipient := message.Phone
		if box != nil {
			var err error
			recipient, err = box.Seal(recipient, "notification-recipient")
			if err != nil {
				return err
			}
		}
		channels := message.Channels
		if channels == nil {
			channels = []string{"sms", "in_app"}
		}
		for _, channel := range channels {
			if len(args) > 0 {
				values.WriteByte(',')
			}
			n := len(args)
			// The inbox entry becomes available with the parent's commit. This
			// records local availability only; it makes no claim about reading.
			state, delivered := "'pending'", "NULL"
			if channel == "in_app" {
				state, delivered = "'delivered'", "now()"
			}
			fmt.Fprintf(&values, "($%d,$%d,$%d,$%d,$%d,%s,%s)", n+1, n+2, n+3, n+4, n+5, state, delivered)
			args = append(args, message.Topic, message.Key, message.FamilyID, channel, recipient)
			if len(args) == 512*5 {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
	return flush()
}

func CareInbox(ctx context.Context, tx *db.Transaction, box *securestore.Box, oldID, incidentID int64) error {
	if !tx.IsMySQL() {
		return nil
	}
	var familyID int64
	var phone string
	err := tx.QueryRow(ctx, "SELECT f.family_id,f.family_phone FROM oldperson o JOIN family f ON f.family_id=o.family_id WHERE o.old_id=$1", oldID).Scan(&familyID, &phone)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return EnqueueChannels(ctx, tx, box, "care", fmt.Sprint(incidentID), familyID, phone, []string{"in_app"})
}
