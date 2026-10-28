package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	dbstore "careold-go/internal/db"
	"careold-go/internal/response"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

const signedDeviceWindow = 5 * time.Minute
const signedDeviceMaxBody = 1 << 20

var signedDeviceNoncePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

type signedDeviceContextKey struct{}
type signedDeviceNonce struct {
	digest    [32]byte
	expiresAt time.Time
}

// Existing unsigned firmware continues to work unless
// CAREOLD_REQUIRE_SIGNED_DEVICE=1. Signatures that are present must verify.
func prepareSignedDeviceRequest(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	sig := strings.TrimSpace(r.Header.Get("X-Device-Signature"))
	if sig == "" && os.Getenv("CAREOLD_REQUIRE_SIGNED_DEVICE") != "1" {
		return r, true
	}
	fail := func() (*http.Request, bool) {
		response.JSON(w, http.StatusUnauthorized, map[string]any{"returnCode": "0004", "message": "设备请求签名无效或已过期"})
		return r, false
	}
	sigBytes, err := hex.DecodeString(sig)
	if err != nil || len(sigBytes) != sha256.Size {
		return fail()
	}
	timestamp := strings.TrimSpace(r.Header.Get("X-Device-Timestamp"))
	nonce := strings.TrimSpace(r.Header.Get("X-Device-Nonce"))
	if !signedDeviceNoncePattern.MatchString(nonce) {
		return fail()
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fail()
	}
	now := time.Now().UTC()
	at := time.Unix(seconds, 0).UTC()
	if at.Before(now.Add(-signedDeviceWindow)) || at.After(now.Add(signedDeviceWindow)) {
		return fail()
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, signedDeviceMaxBody+1))
	if err != nil || len(raw) > signedDeviceMaxBody {
		response.JSON(w, http.StatusRequestEntityTooLarge, map[string]any{"returnCode": "0000", "message": "设备请求体过大或无法读取"})
		return r, false
	}
	defer r.Body.Close()
	var auth struct {
		DeviceNo string `json:"deviceNo"`
		Secret   string `json:"secret"`
	}
	if err := json.Unmarshal(raw, &auth); err != nil || auth.DeviceNo == "" || auth.Secret == "" {
		return fail()
	}
	deviceNo := strings.ToUpper(strings.TrimSpace(auth.DeviceNo))
	bodyHash := sha256.Sum256(raw)
	mac := hmac.New(sha256.New, []byte(auth.Secret))
	_, _ = mac.Write([]byte(deviceNo + ":" + timestamp + ":" + nonce + ":" + hex.EncodeToString(bodyHash[:])))
	if !hmac.Equal(sigBytes, mac.Sum(nil)) {
		return fail()
	}
	// The unique DB key is shared by all replicas; a process-local map is
	// not a replay defence. Commit and nonce insertion are atomic.
	nonceHash := sha256.Sum256([]byte(deviceNo + "\x00" + nonce))
	signed := signedDeviceNonce{digest: nonceHash, expiresAt: at.Add(signedDeviceWindow)}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return r.WithContext(context.WithValue(r.Context(), signedDeviceContextKey{}, signed)), true
}

func persistSignedDeviceNonce(ctx context.Context, tx oldQuerier, deviceID int64) (bool, error) {
	signed, ok := ctx.Value(signedDeviceContextKey{}).(signedDeviceNonce)
	if !ok {
		return true, nil
	}
	_, err := tx.Exec(ctx, `INSERT INTO device_request_nonces(device_id,nonce_digest,expires_at) VALUES($1,$2,$3)`, deviceID, signed.digest[:], signed.expiresAt)
	if err == nil {
		return true, nil
	}
	var myErr *mysqlDriver.MySQLError
	if errors.As(err, &myErr) && myErr.Number == 1062 {
		return false, nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return false, nil
	}
	return false, err
}

func acceptSignedDeviceNonce(w http.ResponseWriter, ctx context.Context, tx oldQuerier, deviceID int64) bool {
	accepted, err := persistSignedDeviceNonce(ctx, tx, deviceID)
	if err != nil {
		response.JSON(w, http.StatusServiceUnavailable, map[string]any{"returnCode": "0000", "message": "设备防重放状态不可用"})
		return false
	}
	if !accepted {
		response.JSON(w, http.StatusConflict, map[string]any{"returnCode": "0004", "message": "设备请求已处理，请使用新的随机数重试"})
	}
	return accepted
}

// Expired records are deleted independently of the request path. SOS and
// telemetry never acknowledge an in-memory queue without a DB commit.
func RunSignedDeviceNonceCleanup(ctx context.Context, pool *dbstore.Database) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			_, err := pool.Exec(cleanupCtx, `DELETE FROM device_request_nonces WHERE expires_at < $1`, time.Now().UTC().Add(-time.Minute))
			cancel()
			if err != nil && ctx.Err() == nil {
				log.Printf("device replay nonce cleanup failed: %v", err)
			}
		}
	}
}
