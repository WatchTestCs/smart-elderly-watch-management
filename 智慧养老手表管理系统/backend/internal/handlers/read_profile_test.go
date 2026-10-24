package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/config"
	"careold-go/internal/csrf"
	"careold-go/internal/health"
	"careold-go/internal/observability"
	"careold-go/internal/ratelimit"
	"careold-go/internal/securestore"
	"careold-go/internal/session"
)

// TestReadPathProfile 管理端读路径画像（opt-in: READ_PROFILE=1，需一次性 TEST_MYSQL_DSN）。
// 种子：经真实写入路径（health.SaveBatch）与直接 SQL 建立 60 设备 / 60 家属 / 200 老人 /
// 120 公告 / 150 站内通知；健康样本 old 1=10000 条（单日上限口径）、old 2=1440 条
// （1/分钟典型日量）。计时走与生产一致的中间件链
// （observability → ratelimit → csrf → auth → 路由，DB 会话 + CSRF 头），
// 每接口 40 次取 p50/p95。statistics 分别测冷（空缓存，两种量级）与热（命中）。
// 另输出冷 statistics 的逐阶段计时（query+scan / unmarshal / aggregate）与
// 各读查询的 EXPLAIN 计划存档，供回归对比与容量评估。
// 目的：管理端每个页面加载都走这些读接口；此前只有写路径画像（health_profile_test.go）。
func TestReadPathProfile(t *testing.T) {
	if os.Getenv("READ_PROFILE") != "1" {
		t.Skip("opt-in: set READ_PROFILE=1")
	}
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Fatal("disposable TEST_MYSQL_DSN required")
	}
	pool := testDatabase(t)
	ctx := context.Background()

	// ---- 种子（直接 SQL：量级数据走批量直插最快；健康样本走真实 SaveBatch）----
	for i := 1; i <= 60; i++ {
		mustExec(t, pool,
			`INSERT INTO family(family_id,family_name,family_relation,family_sex,family_phone,family_address) VALUES($1,$2,'子女','女',$3,'本地')`,
			int64(i), "家属"+strconv.Itoa(i), fmt.Sprintf("139%08d", i))
	}
	for i := 1; i <= 200; i++ {
		mustExec(t, pool,
			`INSERT INTO oldperson(old_id,name,phone,family_id) VALUES($1,$2,$3,$4)`,
			int64(i), "长者"+strconv.Itoa(i), fmt.Sprintf("138%08d", i), int64((i-1)%60+1))
	}
	hash, err := secretHash(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 1; i <= 60; i++ {
		last := now.Add(-10 * time.Second) // 在线
		if i%2 == 0 {
			last = now.Add(-10 * time.Minute) // 离线
		}
		mustExec(t, pool,
			`INSERT INTO device(device_id,device_no,secret,old_id,model,battery,status,bind_time,last_time) VALUES($1,$2,$3,$4,'W8',80,1,$5,$6)`,
			int64(i), fmt.Sprintf("WATCH%03d", i), hash, int64(i), now.AddDate(0, 0, -30), last)
	}
	for i := 1; i <= 120; i++ {
		mustExec(t, pool,
			`INSERT INTO text(title,text_main,time,pic_id) VALUES($1,$2,$3,NULL)`,
			"公告"+strconv.Itoa(i), "内容"+strconv.Itoa(i), now.Add(time.Duration(-i)*time.Hour))
	}
	for i := 1; i <= 150; i++ {
		mustExec(t, pool,
			`INSERT INTO message_outbox(topic,source_key,family_id,channel,recipient,status,created_at) VALUES('announcement',$1,1,'in_app','rec','sent',$2)`,
			fmt.Sprintf("readprof-%d", i), now.Add(time.Duration(-i)*time.Minute))
	}
	// 健康样本：old 1 = 10000 条（单日上限，最坏口径）；old 2 = 1440 条（1/分钟，典型日量）。
	// 均经真实 SaveBatch（同时产出 health_change）。
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	seedSamples := func(oldID int64, total int) {
		for start := 0; start < total; start += 200 {
			end := start + 200
			if end > total {
				end = total
			}
			samples := make([]health.Sample, 0, end-start)
			for i := start; i < end; i++ {
				at := from.Add(time.Duration(i) * time.Second)
				n, err := (health.Sample{
					RecordID:     fmt.Sprintf("readprof-%d-%d", oldID, i),
					Source:       "watch", SourceDevice: "WATCH001",
					Metric: "heart_rate", Semantics: "instant", Unit: "count/min",
					Value: health.Number(72), Start: at, End: at, Version: 1,
				}).Normalize()
				if err != nil {
					t.Fatal(err)
				}
				samples = append(samples, n)
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := health.SaveBatch(ctx, tx, health.Owner{OldID: oldID}, samples); err != nil {
				tx.Rollback(ctx)
				t.Fatal(err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	seedSamples(1, 10000)
	seedSamples(2, 1440)

	// ---- 会话与生产中间件链 ----
	box, err := securestore.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{5}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	sessions := session.NewStore(12 * time.Hour)
	sessions.UseDatabase(pool.SQL(), box)
	// 与生产 env 同名：SESSION_CACHE_MS>0 时启用会话微缓存（读延迟亚毫秒口径）。
	if v := os.Getenv("SESSION_CACHE_MS"); v != "" {
		if ms, e := strconv.Atoi(v); e == nil && ms > 0 {
			sessions.UseSessionCache(time.Duration(ms) * time.Millisecond)
		}
	}
	sess := sessions.Create(session.Principal{Role: auth.RoleAdmin, ID: 1, Phone: "13800000001"})
	if sess == nil {
		t.Fatal("session setup failed")
	}
	buildChain := func(cfg config.Config) (http.Handler, *Handler) {
		h := New(pool, sessions, cfg)
		mux := http.NewServeMux()
		h.Routes(mux)
		var out http.Handler = mux
		out = auth.Middleware(h.Sessions, out)
		out = csrf.Middleware(h.Sessions, out)
		out = ratelimit.New().Middleware(out)
		out = observability.Default.HTTP(out)
		return out, h
	}
	// 与 e2e_chain_bench_test.go 一致：日志写 Discard，不测 IO 成本。
	origLog := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer slog.SetDefault(origLog)

	authed := func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
		r.Header.Set(csrf.HeaderName, sess.CSRFToken)
	}
	warmup := func(chain http.Handler, path string, body []byte) map[string]any {
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		authed(r)
		w := httptest.NewRecorder()
		chain.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s warmup: HTTP %d %s", path, w.Code, w.Body.String())
		}
		var out map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s warmup: %v", path, err)
		}
		return out
	}
	measure := func(name string, chain http.Handler, path string, body []byte, iterations int) {
		durs := make([]float64, 0, iterations)
		for i := 0; i < iterations; i++ {
			r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
			authed(r)
			w := &chainRW{h: make(http.Header)}
			start := time.Now()
			chain.ServeHTTP(w, r)
			d := time.Since(start)
			if w.status != http.StatusOK {
				t.Fatalf("%s: status=%d", name, w.status)
			}
			durs = append(durs, float64(d)/float64(time.Millisecond))
		}
		sort.Float64s(durs)
		t.Logf("READ %-28s n=%d p50=%.2fms p95=%.2fms max=%.2fms",
			name, len(durs), durs[len(durs)/2], durs[len(durs)*95/100], durs[len(durs)-1])
	}

	chain, _ := buildChain(config.Config{HealthCacheBytes: 16 << 20})

	// ---- 逐接口：预热（含响应契约校验）+ 40 次计时 ----
	out := warmup(chain, "/restful/device/list", []byte(`{}`))
	if devices, _ := out["devices"].([]any); len(devices) != 20 || out["total"] != float64(60) || out["onlineCount"] != float64(30) {
		t.Fatalf("device/list unexpected: %#v", out)
	}
	measure("device/list", chain, "/restful/device/list", []byte(`{}`), 40)

	out = warmup(chain, "/restful/notification/list", []byte(`{"after":0,"familyId":1}`))
	if items, _ := out["items"].([]any); len(items) != 100 {
		t.Fatalf("notification/list unexpected: %#v", out)
	}
	measure("notification/list", chain, "/restful/notification/list", []byte(`{"after":0,"familyId":1}`), 40)

	out = warmup(chain, "/restful/text/select", []byte(`{"page":1,"pageSize":20}`))
	if texts, _ := out["texts"].([]any); len(texts) != 20 || out["total"] != float64(120) {
		t.Fatalf("text/select unexpected: %#v", out)
	}
	measure("text/select", chain, "/restful/text/select", []byte(`{"page":1,"pageSize":20}`), 40)

	out = warmup(chain, "/restful/family/select", []byte(`{"page":1,"pageSize":20}`))
	if dtos, _ := out["familyDtos"].([]any); len(dtos) != 20 || out["total"] != float64(200) {
		t.Fatalf("family/select unexpected: %#v", out)
	}
	measure("family/select", chain, "/restful/family/select", []byte(`{"page":1,"pageSize":20}`), 40)

	// 冷：每轮新建 handler（缓存为空 → 必 miss）。生产默认即开启 16MiB 缓存。
	// 分别测最坏口径（old 1，单日上限 10000 条）与典型日量（old 2，1/分钟=1440 条）。
	measureStatisticsCold := func(name string, statBody []byte) {
		coldDurs := make([]float64, 0, 40)
		for i := 0; i < 40; i++ {
			coldChain, _ := buildChain(config.Config{HealthCacheBytes: 16 << 20})
			r := httptest.NewRequest(http.MethodPost, "/restful/health/statistics", bytes.NewReader(statBody))
			authed(r)
			w := &chainRW{h: make(http.Header)}
			start := time.Now()
			coldChain.ServeHTTP(w, r)
			d := time.Since(start)
			if w.status != http.StatusOK {
				t.Fatalf("%s: status=%d", name, w.status)
			}
			coldDurs = append(coldDurs, float64(d)/float64(time.Millisecond))
		}
		sort.Float64s(coldDurs)
		t.Logf("READ %-28s n=%d p50=%.2fms p95=%.2fms max=%.2fms",
			name, len(coldDurs), coldDurs[len(coldDurs)/2], coldDurs[len(coldDurs)*95/100], coldDurs[len(coldDurs)-1])
	}
	statBody10000, err := json.Marshal(map[string]any{
		"oldId": 1, "source": "watch", "sourceDevice": "WATCH001",
		"metric": "heart_rate", "semantics": "instant",
		"from": from, "to": from.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	measureStatisticsCold("health/statistics(cold)", statBody10000)
	out = warmup(chain, "/restful/health/statistics", statBody10000)
	if st, _ := out["statistics"].(map[string]any); st == nil || st["count"] != float64(10000) {
		t.Fatalf("statistics unexpected: %#v", out)
	}
	measure("health/statistics(hot)", chain, "/restful/health/statistics", statBody10000, 40)
	statBodyTypical, err := json.Marshal(map[string]any{
		"oldId": 2, "source": "watch", "sourceDevice": "WATCH001",
		"metric": "heart_rate", "semantics": "instant",
		"from": from, "to": from.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	measureStatisticsCold("health/statistics(cold,1d)", statBodyTypical)
	out = warmup(chain, "/restful/health/statistics", statBodyTypical)
	if st, _ := out["statistics"].(map[string]any); st == nil || st["count"] != float64(1440) {
		t.Fatalf("statistics typical unexpected: %#v", out)
	}

	out = warmup(chain, "/restful/health/changes", []byte(`{"oldId":1,"after":0,"limit":100}`))
	if changes, _ := out["changes"].([]any); len(changes) != 100 || out["hasMore"] != true {
		t.Fatalf("health/changes unexpected: %#v", out)
	}
	measure("health/changes", chain, "/restful/health/changes", []byte(`{"oldId":1,"after":0,"limit":100}`), 40)

	// ---- 冷 statistics 分阶段计时（与 handler_healthStatistics 相同的语句序列）----
	to := from.Add(24 * time.Hour)
	const statSQL = `SELECT payload FROM health_sample
	 WHERE old_id=$1 AND source=$2 AND source_device=$3
	 AND metric=$4 AND semantics=$5 AND NOT deleted
	 AND sample_start<$7
	 AND (sample_start>=$6 OR ($5='interval' AND sample_end>$6))
	 ORDER BY revision LIMIT 10001`
	var queryDur, unmarshalDur, aggDur time.Duration
	var payloadBytes int64
	const phaseN = 10
	for i := 0; i < phaseN; i++ {
		t0 := time.Now()
		rows, err := pool.Query(ctx, statSQL, 1, "watch", "WATCH001", "heart_rate", "instant", from.UnixMicro(), to.UnixMicro())
		if err != nil {
			t.Fatal(err)
		}
		payloads := make([][]byte, 0, 10000)
		for rows.Next() {
			var b []byte
			if err := rows.Scan(&b); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			payloads = append(payloads, b)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		t1 := time.Now()
		samples := make([]health.Sample, 0, len(payloads))
		for _, b := range payloads {
			var s health.Sample
			if err := json.Unmarshal(b, &s); err != nil {
				t.Fatal(err)
			}
			samples = append(samples, s)
		}
		t2 := time.Now()
		if _, err := health.Aggregate(samples, from, to); err != nil {
			t.Fatal(err)
		}
		t3 := time.Now()
		queryDur += t1.Sub(t0)
		unmarshalDur += t2.Sub(t1)
		aggDur += t3.Sub(t2)
		if i == 0 {
			for _, b := range payloads {
				payloadBytes += int64(len(b))
			}
		}
	}
	avg := func(d time.Duration) float64 { return float64(d) / float64(phaseN) / float64(time.Millisecond) }
	t.Logf("PHASE statistics(n=10000) n=%d query+scan=%.2fms unmarshal=%.2fms aggregate=%.2fms payload=%.0fKB",
		phaseN, avg(queryDur), avg(unmarshalDur), avg(aggDur), float64(payloadBytes)/1024)

	// ---- EXPLAIN：各读接口查询计划存档（MySQL 原生方言，$n→?）----
	explain := func(name, query string, args ...any) {
		rows, err := pool.SQL().Query("EXPLAIN "+query, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		cols, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		vals := make([]sql.RawBytes, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			parts := make([]string, 0, len(cols))
			for i, c := range cols {
				if c == "id" || c == "select_type" || c == "partitions" || c == "filtered" {
					continue
				}
				parts = append(parts, c+"="+string(vals[i]))
			}
			t.Logf("EXPLAIN %s: %s", name, strings.Join(parts, " | "))
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}
	explain("statistics", `SELECT payload FROM health_sample
	 WHERE old_id=? AND source=? AND source_device=? AND metric=? AND semantics=? AND NOT deleted
	 AND sample_start<? AND (sample_start>=? OR (?='interval' AND sample_end>?))
	 ORDER BY revision LIMIT 10001`, 1, "watch", "WATCH001", "heart_rate", "instant", to.UnixMicro(), from.UnixMicro(), "instant", from.UnixMicro())
	explain("changes", `SELECT revision,payload FROM health_change WHERE old_id=? AND revision>? ORDER BY revision LIMIT ?`, 1, 0, 101)
	explain("notification/list", `SELECT id,topic,source_key,status,created_at FROM message_outbox WHERE family_id=? AND channel='in_app' AND id>? ORDER BY id LIMIT 100`, 1, 0)
	explain("device/list(count)", `SELECT COUNT(*), COALESCE(SUM(CASE WHEN d.last_time >= CURRENT_TIMESTAMP(6) - INTERVAL 60 SECOND THEN 1 ELSE 0 END),0)
	 FROM device d LEFT JOIN oldperson o ON o.old_id = d.old_id
	 WHERE (?='' OR LOWER(d.device_no) LIKE ? OR LOWER(COALESCE(d.alias,'') || ' ' || COALESCE(o.name,'')) LIKE ?)
	 AND (?='all' OR CASE WHEN d.last_time >= CURRENT_TIMESTAMP(6) - INTERVAL 60 SECOND THEN 'online' ELSE 'offline' END = ?)`, "", "", "", "all", "all")
	explain("text/select(page)", "SELECT text_id, title, text_main, DATE_FORMAT(time,'%Y-%m-%d %H:%i:%s') AS time, pic_id FROM `text` WHERE (?='' OR title LIKE ?) ORDER BY time DESC, text_id DESC LIMIT ? OFFSET ?", "", "%%", 20, 0)
	explain("family/select(page)", `SELECT o.old_id,o.name,f.family_id,f.family_name FROM oldperson o JOIN family f ON o.family_id=f.family_id
	 WHERE (?='' OR f.family_name LIKE '%'||?||'%') ORDER BY o.old_id LIMIT ? OFFSET ?`, "", "", 20, 0)
}
