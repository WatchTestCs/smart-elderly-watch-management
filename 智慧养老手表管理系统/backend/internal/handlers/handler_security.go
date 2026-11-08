package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"suining-care/internal/auth"
	"suining-care/internal/authrepo"
	"suining-care/internal/db"
	"suining-care/internal/health"
	"suining-care/internal/observability"
	"suining-care/internal/response"
	"suining-care/internal/securestore"
)

type factorGenerationKey struct{}

func factorScope(role string, id int64) string {
	return "totp:" + role + ":" + strconv.FormatInt(id, 10)
}
func (h *Handler) loginAudit(r *http.Request, phone, role string, id int64, outcome string) error {
	return h.loginAuditTx(r, phone, role, id, outcome, nil)
}
func (h *Handler) loginAuditTx(r *http.Request, phone, role string, id int64, outcome string, tx *db.Transaction) error {
	observability.Default.Add(`suining_auth_attempts_total{outcome="`+outcome+`"}`, 1)
	if !h.DB.IsMySQL() {
		return nil
	}
	requestID := r.Header.Get("X-Request-ID")
	if len(requestID) > 64 {
		requestID = requestID[:64]
	}
	return authrepo.RecordLoginAttempt(
		r.Context(), h.DB, tx, h.Box.Index("account", phone), h.Box.Index("ip", h.clientIP(r)),
		role, id, outcome, requestID,
	)
}

func (h *Handler) loginAllowed(w http.ResponseWriter, r *http.Request, phone string) bool {
	if !h.DB.IsMySQL() {
		return true
	}
	pair, account, err := authrepo.RecentFailureCounts(
		r.Context(), h.DB, h.Box.Index("account", phone),
		h.Box.Index("ip", h.clientIP(r)), time.Now().Add(-15*time.Minute),
	)
	if internalError(w, err) {
		return false
	}
	if pair >= 10 || account >= 30 {
		w.Header().Set("Retry-After", "900")
		response.JSON(w, 429, map[string]any{"message": "登录尝试过多，请稍后重试"})
		return false
	}
	return true
}
func (h *Handler) factorLogin(w http.ResponseWriter, r *http.Request, m map[string]any, role string, id int64, phone string) bool {
	if !h.DB.IsMySQL() {
		return true
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return false
	}
	defer tx.Rollback(ctx)
	factor, fetchErr := authrepo.LockLoginFactor(ctx, tx, role, id)
	err = fetchErr
	if err != nil && err != pgx.ErrNoRows {
		response.Error500(w, ref())
		return false
	}
	if err == nil && factor.Enabled {
		secret, e := h.Box.Open(factor.Sealed, factorScope(role, id))
		if e != nil {
			response.JSON(w, 503, map[string]any{"message": "安全密钥不可用"})
			return false
		}
		counter, valid := securestore.VerifyTOTP(secret, str(m, "otp"), time.Now(), factor.LastCounter)
		if !valid && str(m, "recoveryCode") != "" {
			var codes []string
			if internalError(w, json.Unmarshal(factor.RecoveryHashes, &codes)) {
				return false
			}
			candidate := health.Digest([]byte(str(m, "recoveryCode")))
			for i, v := range codes {
				if v == candidate {
					codes = append(codes[:i], codes[i+1:]...)
					factor.RecoveryHashes, _ = json.Marshal(codes)
					valid = true
					counter = factor.LastCounter
					break
				}
			}
		}
		if !valid {
			if internalError(w, h.loginAuditTx(r, phone, role, id, "factor_required", tx)) {
				return false
			}
			if internalError(w, tx.Commit(ctx)) {
				return false
			}
			response.JSON(w, 401, map[string]any{"code": "MFA_REQUIRED", "message": "请输入有效的动态验证码或未使用的恢复码"})
			return false
		}
		if internalError(w, authrepo.UpdateLoginFactor(ctx, tx, role, id, counter, factor.RecoveryHashes)) {
			return false
		}
		*r = *r.WithContext(context.WithValue(r.Context(), factorGenerationKey{}, factor.Generation))
	}
	if internalError(w, h.loginAuditTx(r, phone, role, id, "credentials_verified", tx)) {
		return false
	}
	if err = tx.Commit(ctx); internalError(w, err) {
		return false
	}
	return true
}
func (h *Handler) securityFactor(w http.ResponseWriter, r *http.Request) {
	p, ok := principalOr401(w, r)
	if !ok {
		return
	}
	if h.Box == nil {
		response.JSON(w, 503, map[string]any{"message": "须先配置敏感字段加密密钥"})
		return
	}
	var req struct {
		Action   string `json:"action"`
		Password string `json:"password"`
		OTP      string `json:"otp"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(ctx)
	password, passwordErr := authrepo.LockPassword(ctx, tx, p.Role, p.ID)
	if passwordErr != nil || !auth.Matches(req.Password, password) {
		response.JSON(w, 403, map[string]any{"message": "须验证当前密码"})
		return
	}
	scope := factorScope(p.Role, p.ID)
	factor, factorErr := authrepo.LockFactor(ctx, tx, p.Role, p.ID)
	err = factorErr
	if err != nil && err != pgx.ErrNoRows {
		response.Error500(w, ref())
		return
	}
	result := map[string]any{}
	switch req.Action {
	case "setup":
		if factor.Enabled {
			conflict409(w, "已启用双因素认证，不能覆盖现有密钥")
			return
		}
		result, err = h.setupFactor(ctx, tx, p.Role, p.ID, scope)
	case "confirm", "disable":
		if err == pgx.ErrNoRows || (req.Action == "confirm" && (factor.Enabled || time.Now().After(factor.PendingUntil))) || (req.Action == "disable" && !factor.Enabled) {
			conflict409(w, "双因素认证状态不允许此操作")
			return
		}
		secret, e := h.Box.Open(factor.Sealed, scope)
		if internalError(w, e) {
			return
		}
		counter, valid := securestore.VerifyTOTP(secret, req.OTP, time.Now(), factor.LastCounter)
		if !valid {
			response.JSON(w, 403, map[string]any{"message": "动态验证码无效或已使用"})
			return
		}
		if req.Action == "disable" {
			err = authrepo.DeleteFactor(ctx, tx, p.Role, p.ID)
		} else {
			codes, hashes, codeErr := newRecoveryCodes()
			if internalError(w, codeErr) {
				return
			}
			b, _ := json.Marshal(hashes)
			err = authrepo.EnableFactor(ctx, tx, p.Role, p.ID, counter, b)
			result["recoveryCodes"] = codes
		}
		result["reloginRequired"] = true
	default:
		write400(w, "操作须为 setup、confirm 或 disable")
		return
	}
	if err == nil && req.Action != "setup" && h.Cfg.SessionBackend == "database" {
		err = authrepo.RevokeDatabaseSessions(ctx, tx, p.Role, p.ID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if internalError(w, err) {
		return
	}
	if req.Action != "setup" && h.Cfg.SessionBackend != "database" {
		// The factor change is committed: cancellation must not retain the old
		// memory sessions. Database mode revoked within the same transaction.
		if err = h.Sessions.RevokeContext(context.WithoutCancel(ctx), p.Role, p.ID); err != nil {
			response.JSON(w, 503, map[string]any{"message": "认证状态已更新，但会话撤销需重试"})
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	response.OK(w, result)
}

func (h *Handler) securityLoginAudit(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.careSupervisor(w, r); !ok {
		return
	}
	var req struct {
		After int64 `json:"after"`
	}
	if !decodeCapability(w, r, &req) {
		return
	}
	if req.After < 0 {
		write400(w, "游标无效")
		return
	}
	items, err := authrepo.ListLoginAudits(r.Context(), h.DB, req.After)
	if internalError(w, err) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}

func (h *Handler) setupFactor(ctx context.Context, tx *db.Transaction, role string, actorID int64, scope string) (map[string]any, error) {
	seed := make([]byte, 20)
	if _, err := rand.Read(seed); err != nil {
		return nil, err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(seed)
	sealed, err := h.Box.Seal(secret, scope)
	if err == nil {
		err = authrepo.SetupFactor(ctx, tx, role, actorID, sealed, time.Now().Add(10*time.Minute), uuid.NewString())
	}
	result := map[string]any{"secret": secret, "uri": "otpauth://totp/" + url.PathEscape("岁宁:"+role+":"+strconv.FormatInt(actorID, 10)) + "?secret=" + secret + "&issuer=" + url.QueryEscape("岁宁") + "&algorithm=SHA1&digits=6&period=30"}
	return result, err
}

func newRecoveryCodes() ([]string, []string, error) {
	codes, hashes := []string{}, []string{}
	for range 8 {
		b := make([]byte, 12)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, err
		}
		code := hex.EncodeToString(b)
		codes = append(codes, code)
		hashes = append(hashes, health.Digest([]byte(code)))
	}
	return codes, hashes, nil
}
