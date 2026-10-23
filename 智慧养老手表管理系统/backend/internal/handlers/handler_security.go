package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/db"
	"careold-go/internal/health"
	"careold-go/internal/observability"
	"careold-go/internal/response"
	"careold-go/internal/securestore"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	q := "INSERT INTO login_audit(account_hash,ip_hash,role,actor_id,outcome,request_id) VALUES($1,$2,$3,$4,$5,$6)"
	args := []any{h.Box.Index("account", phone), h.Box.Index("ip", ip), role, id, outcome, r.Header.Get("X-Request-ID")}
	if tx != nil {
		_, err := tx.Exec(r.Context(), q, args...)
		return err
	}
	_, err := h.DB.Exec(r.Context(), q, args...)
	return err
}
func (h *Handler) loginAllowed(w http.ResponseWriter, r *http.Request, phone string) bool {
	if !h.DB.IsMySQL() {
		return true
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	var failures int
	err := h.DB.QueryRow(r.Context(), "SELECT count(*) FROM login_audit WHERE account_hash=$1 AND ip_hash=$2 AND outcome IN ('invalid_credentials','factor_required') AND created_at>$3", h.Box.Index("account", phone), h.Box.Index("ip", ip), time.Now().Add(-15*time.Minute)).Scan(&failures)
	if err != nil {
		response.Error500(w, ref())
		return false
	}
	if failures >= 10 {
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
	if err != nil {
		response.Error500(w, ref())
		return false
	}
	defer tx.Rollback(ctx)
	var sealed string
	var enabled bool
	var last int64
	var hashes []byte
	var generation string
	err = tx.QueryRow(ctx, "SELECT secret,enabled,last_counter,recovery_hashes,generation FROM auth_factor WHERE role=$1 AND actor_id=$2 FOR UPDATE", role, id).Scan(&sealed, &enabled, &last, &hashes, &generation)
	if err != nil && err != pgx.ErrNoRows {
		response.Error500(w, ref())
		return false
	}
	if err == nil && enabled {
		secret, e := h.Box.Open(sealed, factorScope(role, id))
		if e != nil {
			response.JSON(w, 503, map[string]any{"message": "安全密钥不可用"})
			return false
		}
		counter, valid := securestore.VerifyTOTP(secret, str(m, "otp"), time.Now(), last)
		if !valid && str(m, "recoveryCode") != "" {
			var codes []string
			if json.Unmarshal(hashes, &codes) != nil {
				response.Error500(w, ref())
				return false
			}
			candidate := health.Digest([]byte(str(m, "recoveryCode")))
			for i, v := range codes {
				if v == candidate {
					codes = append(codes[:i], codes[i+1:]...)
					hashes, _ = json.Marshal(codes)
					valid = true
					counter = last
					break
				}
			}
		}
		if !valid {
			if h.loginAuditTx(r, phone, role, id, "factor_required", tx) != nil {
				response.Error500(w, ref())
				return false
			}
			if tx.Commit(ctx) != nil {
				response.Error500(w, ref())
				return false
			}
			response.JSON(w, 401, map[string]any{"code": "MFA_REQUIRED", "message": "请输入有效的动态验证码或未使用的恢复码"})
			return false
		}
		if _, err = tx.Exec(ctx, "UPDATE auth_factor SET last_counter=$3,recovery_hashes=$4 WHERE role=$1 AND actor_id=$2", role, id, counter, json.RawMessage(hashes)); err != nil {
			response.Error500(w, ref())
			return false
		}
		*r = *r.WithContext(context.WithValue(r.Context(), factorGenerationKey{}, generation))
	}
	if h.loginAuditTx(r, phone, role, id, "credentials_verified", tx) != nil {
		response.Error500(w, ref())
		return false
	}
	if err = tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)
	query := "SELECT password FROM users WHERE user_id=$1 FOR UPDATE"
	if p.Role == auth.RoleFamily {
		query = "SELECT family_password FROM family WHERE family_id=$1 FOR UPDATE"
	}
	var password string
	if err = tx.QueryRow(ctx, query, p.ID).Scan(&password); err != nil || !auth.Matches(req.Password, password) {
		response.JSON(w, 403, map[string]any{"message": "须验证当前密码"})
		return
	}
	scope := factorScope(p.Role, p.ID)
	var sealed string
	var enabled bool
	var until time.Time
	var last int64
	err = tx.QueryRow(ctx, "SELECT secret,enabled,pending_until,last_counter FROM auth_factor WHERE role=$1 AND actor_id=$2 FOR UPDATE", p.Role, p.ID).Scan(&sealed, &enabled, &until, &last)
	if err != nil && err != pgx.ErrNoRows {
		response.Error500(w, ref())
		return
	}
	result := map[string]any{}
	switch req.Action {
	case "setup":
		if enabled {
			conflict409(w, "已启用双因素认证，不能覆盖现有密钥")
			return
		}
		seed := make([]byte, 20)
		if _, err = rand.Read(seed); err != nil {
			response.Error500(w, ref())
			return
		}
		secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(seed)
		sealed, err = h.Box.Seal(secret, scope)
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO auth_factor(role,actor_id,secret,pending_until,recovery_hashes,generation) VALUES($1,$2,$3,$4,'[]',$5) ON CONFLICT(role,actor_id) DO UPDATE SET secret=EXCLUDED.secret,pending_until=EXCLUDED.pending_until,last_counter=-1,generation=EXCLUDED.generation`, p.Role, p.ID, sealed, time.Now().Add(10*time.Minute), uuid.NewString())
		}
		result = map[string]any{"secret": secret, "uri": "otpauth://totp/" + url.PathEscape("岁宁:"+p.Role+":"+strconv.FormatInt(p.ID, 10)) + "?secret=" + secret + "&issuer=" + url.QueryEscape("岁宁") + "&algorithm=SHA1&digits=6&period=30"}
	case "confirm", "disable":
		if err == pgx.ErrNoRows || (req.Action == "confirm" && (enabled || time.Now().After(until))) || (req.Action == "disable" && !enabled) {
			conflict409(w, "双因素认证状态不允许此操作")
			return
		}
		secret, e := h.Box.Open(sealed, scope)
		if e != nil {
			response.Error500(w, ref())
			return
		}
		counter, valid := securestore.VerifyTOTP(secret, req.OTP, time.Now(), last)
		if !valid {
			response.JSON(w, 403, map[string]any{"message": "动态验证码无效或已使用"})
			return
		}
		if req.Action == "disable" {
			_, err = tx.Exec(ctx, "DELETE FROM auth_factor WHERE role=$1 AND actor_id=$2", p.Role, p.ID)
		} else {
			codes, hashes := []string{}, []string{}
			for range 8 {
				b := make([]byte, 12)
				if _, err = rand.Read(b); err != nil {
					response.Error500(w, ref())
					return
				}
				code := hex.EncodeToString(b)
				codes = append(codes, code)
				hashes = append(hashes, health.Digest([]byte(code)))
			}
			b, _ := json.Marshal(hashes)
			_, err = tx.Exec(ctx, "UPDATE auth_factor SET enabled=true,last_counter=$3,recovery_hashes=$4 WHERE role=$1 AND actor_id=$2", p.Role, p.ID, counter, json.RawMessage(b))
			result["recoveryCodes"] = codes
		}
		result["reloginRequired"] = true
	default:
		write400(w, "操作须为 setup、confirm 或 disable")
		return
	}
	if err == nil && req.Action != "setup" && h.Cfg.SessionBackend == "database" {
		_, err = tx.Exec(ctx, "DELETE FROM web_session WHERE role=$1 AND actor_id=$2", p.Role, p.ID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if req.Action != "setup" && h.Cfg.SessionBackend != "database" {
		if err = h.Sessions.Revoke(p.Role, p.ID); err != nil {
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
	rows, err := h.DB.Query(r.Context(), "SELECT id,role,actor_id,outcome,request_id,created_at FROM login_audit WHERE id>$1 ORDER BY id LIMIT 100", req.After)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, actor int64
		var role, outcome, rid string
		var at time.Time
		if rows.Scan(&id, &role, &actor, &outcome, &rid, &at) != nil {
			response.Error500(w, ref())
			return
		}
		items = append(items, map[string]any{"id": id, "role": role, "actorId": actor, "outcome": outcome, "requestId": rid, "createdAt": at})
	}
	if !rowsOK(w, rows) {
		return
	}
	response.OK(w, map[string]any{"items": items})
}
