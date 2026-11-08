// 会话、登录、初始化和桌面端控制接口。
package handlers

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"

	"suining-care/internal/auth"
	"suining-care/internal/csrf"
	"suining-care/internal/response"
	"suining-care/internal/session"
	"suining-care/internal/userphone"
)

func (h *Handler) routeAuth(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/session", h.sessionInfo)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("POST /api/v1/setup", h.setup)
	mux.HandleFunc("POST /api/v1/desktop/exit", h.desktopExit)
}

// timingDummyHash 是一条与本哈希同 cost（12）的无效 bcrypt 记录：手机号在
// users/family 两表都不存在时也执行一次口令比较，使"账号是否存在"无法通过
// 响应时间区分。它不是任何账号的凭据。
const timingDummyHash = "$2a$12$S1k/y1BsWtNAy/eeN.zaleT.hNjDpyRQ/1S7MCXu5Y8GN61EoKbMK"

// MountDesktopExit allows closing the desktop even before the database is ready.
func (h *Handler) MountDesktopExit(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/desktop/exit", h.desktopExit)
}

// MountLocalLogout keeps session cleanup available without a database.
func (h *Handler) MountLocalLogout(mux *http.ServeMux) {
	local := http.NewServeMux()
	local.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.Handle("POST /api/v1/auth/logout", csrf.Middleware(h.Sessions, auth.Middleware(h.Sessions, local)))
}

// sessionInfo 处理 GET /api/v1/auth/session：公开；无会话则建匿名会话。
func (h *Handler) sessionInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var count int64
	if internalError(w, h.DB.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)) {
		return
	}

	sess := h.Sessions.FromRequest(r)
	if sess == nil {
		sess = h.Sessions.CreateContext(r.Context(), session.Principal{})
		if sess == nil {
			response.JSON(w, 503, map[string]any{"message": "会话存储不可用"})
			return
		}
		session.SetCookie(w, sess.ID, h.CookieSecure)
	}
	authenticated := sess.Principal.Role == auth.RoleAdmin || sess.Principal.Role == auth.RoleFamily
	account := map[string]any{}
	if authenticated {
		p := sess.Principal
		account = map[string]any{
			"role": p.Role, "id": p.ID, "phone": p.Phone,
			"oldId": p.OldID, "displayName": p.DisplayName,
		}
		if p.Role == auth.RoleAdmin {
			duty, ok := h.careDuty(w, r, p.ID)
			if !ok {
				return
			}
			account["duty"] = duty
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"authenticated": authenticated,
		"setupRequired": count == 0,
		"account":       account,
		"csrfToken":     sess.CSRFToken,
		"csrfHeader":    "X-CSRF-TOKEN",
	})
}

// Try the administrator account first when both roles share a phone number.
var loginAccounts = [...]struct{ role, query string }{
	{auth.RoleAdmin, `SELECT user_id,user_name,user_phone,password,NULL,user_phone_cipher,user_phone_lookup,user_phone_protected
 FROM users WHERE (user_phone_protected=false AND user_phone=$1) OR (user_phone_protected=true AND user_phone_lookup=$2)`},
	{auth.RoleFamily, `SELECT f.family_id, f.family_name, f.family_phone, f.family_password, o.old_id, NULL, NULL, false
  FROM family f LEFT JOIN oldperson o ON o.family_id=f.family_id
  WHERE f.family_phone=$1 ORDER BY o.old_id LIMIT 1`},
}

// login 处理 POST /api/v1/auth/login：公开；先查 users（管理员），再查 family（家属）。
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone := str(m, "phone")
	password := str(m, "password")
	if !auth.ValidPhone(phone) || password == "" || len(password) > 72 {
		response.JSON(w, http.StatusBadRequest, map[string]any{"code": "INVALID_LOGIN_INPUT", "message": "请输入有效手机号和密码"})
		return
	}

	if !h.loginAllowed(w, r, phone) {
		return
	}
	lookup, err := userphone.SearchIndex(r.Context(), h.DB, h.Box, phone)
	if internalError(w, err) {
		return
	}
	compared := false
	for _, account := range loginAccounts {
		var id int64
		var name, storedPhone, hash pgtype.Text
		var oldID pgtype.Int8
		var protected userphone.Record
		args := []any{phone}
		if account.role == auth.RoleAdmin {
			args = append(args, lookup)
		}
		err := h.DB.QueryRow(r.Context(), account.query, args...).Scan(&id, &name, &storedPhone, &hash, &oldID, &protected.Cipher, &protected.Lookup, &protected.Protected)
		if isNotFound(err) {
			continue
		}
		if internalError(w, err) {
			return
		}
		if account.role == auth.RoleAdmin {
			if storedPhone.Valid {
				protected.Legacy = &storedPhone.String
			}
			opened, err := protected.OpenStored(h.Box, id)
			if internalError(w, err) {
				return
			}
			storedPhone = pgtype.Text{}
			if opened != nil {
				storedPhone = pgtype.Text{String: *opened, Valid: true}
			}
		}
		compared = true
		if !auth.Matches(password, hash.String) {
			continue
		}
		if account.role == auth.RoleFamily && !oldID.Valid {
			response.JSON(w, http.StatusForbidden, map[string]any{"code": "ACCOUNT_UNLINKED", "message": "该家属账号尚未关联老人，请联系管理员"})
			return
		}
		h.finishAccountLogin(w, r, m, session.Principal{
			Role: account.role, ID: id, Phone: storedPhone.String, OldID: oldID.Int64, DisplayName: name.String,
		}, phone)
		return
	}

	if !compared {
		// 两次查询都未命中：补一次等价 bcrypt 比较，消除账号枚举计时侧信道。
		_ = auth.Matches(password, timingDummyHash)
	}
	if internalError(w, h.loginAudit(r, phone, "", 0, "invalid_credentials")) {
		return
	}
	response.JSON(w, http.StatusUnauthorized, map[string]any{"code": "INVALID_CREDENTIALS", "message": "手机号或密码不正确"})
}

// renewSession 登录建会话：销毁请求 cookie 带来的旧会话（防 session fixation），建新会话并写 cookie。
func (h *Handler) renewSession(w http.ResponseWriter, r *http.Request, p session.Principal) bool {
	p.FactorGeneration, _ = r.Context().Value(factorGenerationKey{}).(string)
	if c, err := r.Cookie(session.CookieName); err == nil && c.Value != "" {
		if !h.Sessions.DestroyContext(r.Context(), c.Value) {
			response.JSON(w, 503, map[string]any{"message": "会话存储不可用"})
			return false
		}
	}
	sess := h.Sessions.CreateContext(r.Context(), p)
	if sess == nil {
		response.JSON(w, 503, map[string]any{"message": "会话存储不可用"})
		return false
	}
	if h.DB != nil && h.loginAudit(r, p.Phone, p.Role, p.ID, "success") != nil {
		// The new session has already been persisted. Compensate even if the
		// request was cancelled; the store still imposes its three-second bound.
		h.Sessions.DestroyContext(context.WithoutCancel(r.Context()), sess.ID)
		response.Error500(w, ref())
		return false
	}
	session.SetCookie(w, sess.ID, h.CookieSecure)
	return true
}

// logout 处理 POST /api/v1/auth/logout：需登录（中间件鉴权），销毁会话并清 cookie。
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	sess := session.FromContext(r)
	if sess == nil {
		sess = h.Sessions.FromRequest(r)
	}
	if sess != nil {
		if !h.Sessions.DestroyContext(r.Context(), sess.ID) {
			response.JSON(w, 503, map[string]any{"message": "会话撤销未完成"})
			return
		}
	}
	session.ClearCookie(w, h.CookieSecure)
	response.OK(w, nil)
}

// setup 处理 POST /api/v1/setup：公开 + 桌面 token；一次性初始化管理员账号。
func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	if !h.checkDesktopToken(w, r) {
		return
	}
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone := str(m, "phone")
	name := str(m, "name")
	password := str(m, "password")

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if internalError(w, err) {
		return
	}
	defer tx.Rollback(ctx)

	// 行锁串行化，防止并发重复初始化。
	var metaID int64
	if internalError(w, tx.QueryRow(ctx, "SELECT id FROM app_metadata WHERE id=1 FOR UPDATE").Scan(&metaID)) {
		return
	}
	var count int64
	if internalError(w, tx.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)) {
		return
	}
	if count != 0 {
		response.JSON(w, http.StatusConflict, map[string]any{"message": "管理员已创建"})
		return
	}
	if !auth.ValidPhone(phone) || strings.TrimSpace(name) == "" ||
		utf8.RuneCountInString(name) > 10 || !auth.ValidNewPassword(password) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"message": "请检查姓名、手机号和密码"})
		return
	}
	hashed, err := auth.Hash(password)
	if internalError(w, err) {
		return
	}
	if h.Box == nil {
		// Staged rollout: without a configured encryption key, keep the original
		// plaintext write so first-run setup stays available on deployments that
		// have not enabled administrator phone protection yet. Existing legacy
		// rows remain readable through the unprotected login branch.
		if _, err := tx.Exec(ctx,
			"INSERT INTO users(user_name,user_phone,password,role) VALUES($1,$2,$3,'1')",
			name, phone, hashed); internalError(w, err) {
			return
		}
	} else {
		var id int64
		if err := tx.QueryRow(ctx,
			"INSERT INTO users(user_name,password,role) VALUES($1,$2,'1') RETURNING user_id",
			name, hashed).Scan(&id); internalError(w, err) {
			return
		}
		protected, err := userphone.Protect(h.Box, id, phone)
		if internalError(w, err) {
			return
		}
		if internalError(w, userphone.BindLookupKey(ctx, tx, h.Box)) {
			return
		}
		if _, err = tx.Exec(ctx, `UPDATE users SET user_phone_cipher=$1,user_phone_lookup=$2,user_phone_protected=true WHERE user_id=$3`,
			protected.Ciphertext, protected.Lookup, id); internalError(w, err) {
			return
		}
	}
	if internalError(w, tx.Commit(ctx)) {
		return
	}
	response.OK(w, nil)
}

// desktopExit 处理 POST /api/v1/desktop/exit：公开 + 桌面 token；通过后延迟退出进程。
func (h *Handler) desktopExit(w http.ResponseWriter, r *http.Request) {
	if !h.checkDesktopToken(w, r) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"closing": true})
	go func() {
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}()
}

// checkDesktopToken 校验 X-Suining-Desktop header（常量时间比较）；失败写 403 空 body。
func (h *Handler) checkDesktopToken(w http.ResponseWriter, r *http.Request) bool {
	token := h.Cfg.DesktopToken
	got := r.Header.Get("X-Suining-Desktop")
	if token == "" || got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
		w.WriteHeader(http.StatusForbidden)
		return false
	}
	return true
}

func (h *Handler) finishAccountLogin(w http.ResponseWriter, r *http.Request, m map[string]any, p session.Principal, phone string) {
	if !h.factorLogin(w, r, m, p.Role, p.ID, phone) {
		return
	}
	if !h.renewSession(w, r, p) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"authenticated": true})
}
