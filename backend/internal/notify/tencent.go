package notify

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func TencentSMS(ctx context.Context, client *http.Client, appID, key, phone string) error {
	if !regexp.MustCompile(`^\d+$`).MatchString(appID) || key == "" || !regexp.MustCompile(`^1\d{10}$`).MatchString(phone) {
		return fmt.Errorf("SMS configuration or recipient invalid")
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	nonce := binary.BigEndian.Uint64(random[:])%900000 + 100000
	ts := time.Now().Unix()
	material := "appkey=" + key + "&random=" + strconv.FormatUint(nonce, 10) + "&time=" + strconv.FormatInt(ts, 10) + "&mobile=" + phone
	sum := sha256.Sum256([]byte(material))
	body := map[string]any{"tel": map[string]any{"nationcode": "86", "mobile": phone}, "type": 0, "msg": "岁宁有新的照护公告，请登录查看详情", "sig": hex.EncodeToString(sum[:]), "time": ts, "extend": "", "ext": ""}
	data, _ := json.Marshal(body)
	u := "https://yun.tim.qq.com/v5/tlssmssvr/sendsms?sdkappid=" + appID + "&random=" + strconv.FormatUint(nonce, 10)
	r, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(data))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		Result int `json:"result"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil || resp.StatusCode != 200 || out.Result != 0 {
		return fmt.Errorf("SMS provider not accepted")
	}
	return nil
}
