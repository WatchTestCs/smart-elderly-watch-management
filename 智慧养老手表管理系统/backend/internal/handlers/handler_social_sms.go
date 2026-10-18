package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)
// SmsUtil 等价实现：goroutine + 缓冲 channel(100)，after-commit 语义
// ---------------------------------------------------------------------------

var (
	smsOnce    sync.Once
	smsQueue   chan string
	smsAppIDRe = regexp.MustCompile(`^\d+$`)
	smsPhoneRe = regexp.MustCompile(`^1\d{10}$`)
)

// smsEnqueue 未配置 appId/appKey 或手机号非法时静默跳过；队列满则丢弃记 warn。
func smsEnqueue(phone string) {
	appID := os.Getenv("TENCENT_SMS_APPID")
	appKey := os.Getenv("TENCENT_SMS_APPKEY")
	if !smsAppIDRe.MatchString(appID) || appKey == "" || !smsPhoneRe.MatchString(phone) {
		return
	}
	smsOnce.Do(func() {
		smsQueue = make(chan string, 100)
		go smsWorker(appID, appKey)
	})
	select {
	case smsQueue <- phone:
	default:
		log.Println("suining sms: queue full, dropped")
	}
}

func smsWorker(appID, appKey string) {
	client := &http.Client{Timeout: 15 * time.Second}
	for phone := range smsQueue {
		sendTencentSMS(client, appID, appKey, phone)
	}
}

func sendTencentSMS(client *http.Client, appID, appKey, phone string) {
	nonce := rand.Int63n(900000) + 100000
	ts := time.Now().Unix()
	material := "appkey=" + appKey + "&random=" + strconv.FormatInt(nonce, 10) +
		"&time=" + strconv.FormatInt(ts, 10) + "&mobile=" + phone
	sum := sha256.Sum256([]byte(material))
	body := map[string]any{
		"tel":    map[string]any{"nationcode": "86", "mobile": phone},
		"type":   0,
		"msg":    "岁宁有新的照护公告，请登录查看详情",
		"sig":    hex.EncodeToString(sum[:]),
		"time":   ts,
		"extend": "",
		"ext":    "",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return
	}
	url := "https://yun.tim.qq.com/v5/tlssmssvr/sendsms?sdkappid=" + appID + "&random=" + strconv.FormatInt(nonce, 10)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("suining sms: send failed:", err)
		return
	}
	defer resp.Body.Close()
	var out struct {
		Result int `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || resp.StatusCode != 200 || out.Result != 0 {
		log.Println("suining sms: not accepted")
	}
}
