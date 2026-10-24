package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"careold-go/internal/auth"
)

// TestLoginPhaseProfile 登录路径延迟画像（opt-in: LOGIN_PROFILE=1）。
// 登录延迟由设计主导（BCrypt cost 12 的 CPU 成本是抗爆破约束）；
// 本画像量化 HTTP 全程与其中 DB/BCrypt 的占比，确认无多余往返。
func TestLoginPhaseProfile(t *testing.T) {
	if os.Getenv("LOGIN_PROFILE") != "1" {
		t.Skip("opt-in: set LOGIN_PROFILE=1")
	}
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Fatal("disposable TEST_MYSQL_DSN required")
	}
	pool := testDatabase(t)
	ctx := context.Background()
	hash, err := auth.Hash("LocalTestOnly1234")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users(user_name,password,role,user_phone) VALUES($1,$2,'ADMIN',$3)`, "prof-admin", hash, "13800000001"); err != nil {
		t.Fatal(err)
	}
	handler := testHandler(pool)
	server := httptest.NewServer(http.HandlerFunc(handler.login))
	defer server.Close()
	client := &http.Client{Timeout: 30 * time.Second}
	defer client.CloseIdleConnections()

	latencies := make([]float64, 0, 24)
	for i := 0; i < 24; i++ { // 限流 30/10min，留余量
		body := bytes.NewReader([]byte(`{"phone":"13800000001","password":"LocalTestOnly1234"}`))
		req, err := http.NewRequest(http.MethodPost, server.URL, body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		at := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		ms := float64(time.Since(at)) / float64(time.Millisecond)
		if resp.StatusCode != http.StatusOK || !bytes.Contains(buf[:n], []byte("1111")) {
			t.Fatalf("login %d failed: status=%d body=%s", i, resp.StatusCode, buf[:n])
		}
		latencies = append(latencies, ms)
	}
	sort.Float64s(latencies)
	// BCrypt 单独成本（10 次）。
	var bcryptTotal time.Duration
	for i := 0; i < 10; i++ {
		at := time.Now()
		if !auth.Matches("LocalTestOnly1234", hash) {
			t.Fatal("bcrypt mismatch")
		}
		bcryptTotal += time.Since(at)
	}
	// 裸往返参考（20 次）。
	var rtTotal time.Duration
	for i := 0; i < 20; i++ {
		at := time.Now()
		if err := pool.QueryRow(ctx, "SELECT 1").Scan(&rtTotal); err != nil {
			t.Fatal(err)
		}
		rtTotal += time.Since(at)
	}
	t.Logf("LOGIN p50=%.1fms p95=%.1fms max=%.1fms n=%d | bcrypt avg=%.1fms | bare RT avg=%.2fms",
		latencies[len(latencies)/2], latencies[len(latencies)*95/100], latencies[len(latencies)-1],
		len(latencies), float64(bcryptTotal)/10/float64(time.Millisecond), float64(rtTotal)/20/float64(time.Millisecond))
}
