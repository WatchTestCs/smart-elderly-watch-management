package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func (d *Worker) deliver(ctx context.Context, id int64, topic, key, channel, phone string) error {
	if channel == "sms" && os.Getenv("TENCENT_SMS_APPID") != "" {
		return TencentSMS(ctx, d.Client, os.Getenv("TENCENT_SMS_APPID"), os.Getenv("TENCENT_SMS_APPKEY"), phone)
	}
	if d.URL == "" || d.Token == "" {
		return fmt.Errorf("gateway unavailable")
	}
	if !strings.HasPrefix(d.URL, "https://") && os.Getenv("NOTIFY_ALLOW_HTTP_TEST") != "1" {
		return fmt.Errorf("gateway requires HTTPS")
	}
	b, _ := json.Marshal(map[string]any{"type": topic, "sourceId": key, "channel": channel, "recipient": phone, "notificationId": id})
	r, err := http.NewRequestWithContext(ctx, "POST", d.URL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+d.Token)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Idempotency-Key", fmt.Sprintf("message-%d", id))
	resp, err := d.Client.Do(r)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gateway not accepted")
	}
	return nil
}
