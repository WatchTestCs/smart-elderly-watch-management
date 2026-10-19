// 登录 / session / 初始化 / 桌面端：LoginController、SetupController、DesktopController。
package handlers

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"

	"careold-go/internal/auth"
	"careold-go/internal/csrf"
	"careold-go/internal/response"
	"careold-go/internal/session"
)

func (h *Handler) routeAuth(mux *http.ServeMux) {
	mux.HandleFunc("GET /restful/userLogin/session", h.sessionInfo)
	mux.HandleFunc("POST /restful/userLogin/login", h.login)
	mux.HandleFunc("POST /restful/userLogin/logout", h.logout)
	mux.HandleFunc("POST /restful/setup", h.setup)
	mux.HandleFunc("POST /restful/desktop/exit", h.desktopExit)
}

// MountDesktopExit allows closing the desktop even before the database is ready.
func (h *Handler) MountDesktopExit(mux *http.ServeMux) {
	mux.HandleFunc("POST /restful/desktop/exit", h.desktopExit)
}

// MountLocalLogout keeps session cleanup available without a database.
func (h *Handler) MountLocalLogout(mux *http.ServeMux) {
	local := http.NewServeMux()
	local.HandleFunc("POST /restful/userLogin/logout", h.logout)
	mux.Handle("POST /restful/userLogin/logout", csrf.Middleware(h.Sessions, auth.Middleware(h.Sessions, local)))
}

// sessionInfo 复刻 GET /restful/userLogin/session：公开；无会话则建匿名会话。
func (h *Handler) sessionInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var count int64
	if err := h.DB.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		response.Error500(w, ref())
		return
	}

	sess := h.Sessions.FromRequest(r)
	if sess == nil {
		sess = h.Sessions.Create(session.Principal{})
		session.SetCookie(w, sess.ID)
	}
	authenticated := sess.Principal.Role == auth.RoleAdmin || sess.Principal.Role == auth.RoleFamily
	account := map[string]any{}
	if authenticated {
		p := sess.Principal
		account = map[string]any{
			"role": p.Role, "id": p.ID, "phone": p.Phone,
			"oldId": p.OldID, "displayName": p.DisplayName,
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

// login 复刻 POST /restful/userLogin/login：公开；先查 users（管理员），再查 family（家属）。
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	m, ok := bodyMap(w, r)
	if !ok {
		return
	}
	phone := str(m, "phone")
	password := str(m, "password")
	if !auth.ValidPhone(phone) || password == "" || len(password) > 72 {
		response.Fail(w, "请输入手机号和密码")
		return
	}

	ctx := r.Context()

	// 1. users 表（管理员）
	var (
		uid    int64
		uName  pgtype.Text
		uPass  pgtype.Text
		uRole  pgtype.Text
		uPhone pgtype.Text
	)
	err := h.DB.QueryRow(ctx,
		`SELECT user_id, user_name, password, role, user_phone FROM users WHERE user_phone=$1`,
		phone).Scan(&uid, &uName, &uPass, &uRole, &uPhone)
	if err != nil && !isNotFound(err) {
		response.Error500(w, ref())
		return
	}
	if err == nil && auth.Matches(password, uPass.String) {
		h.renewSession(w, r, session.Principal{
			Role: auth.RoleAdmin, ID: uid, Phone: uPhone.String, DisplayName: uName.String,
		})
		response.OK(w, map[string]any{
			"users1": map[string]any{
				"userId": uid, "userName": uName.String, "userPhone": uPhone.String,
				"role": uRole.String, "password": nil,
			},
		})
		return
	}

	// 2. family 表（家属）
	var (
		fid     int64
		fName   pgtype.Text
		fRel    pgtype.Text
		fPhone  pgtype.Text
		fSex    pgtype.Text
		fAddr   pgtype.Text
		fTouPic pgtype.Text
		fPass   pgtype.Text
		oldID   pgtype.Int8
		oldName pgtype.Text
	)
	err = h.DB.QueryRow(ctx, `
		SELECT f.family_id, f.family_name, f.family_relation, f.family_phone, f.family_sex,
		       f.family_address, f.family_tou_pic, f.family_password, o.old_id, o.name
		FROM family f LEFT JOIN oldperson o ON o.family_id = f.family_id
		WHERE f.family_phone=$1
		ORDER BY o.old_id LIMIT 1`, phone).
		Scan(&fid, &fName, &fRel, &fPhone, &fSex, &fAddr, &fTouPic, &fPass, &oldID, &oldName)
	if err != nil && !isNotFound(err) {
		response.Error500(w, ref())
		return
	}
	if err == nil && auth.Matches(password, fPass.String) {
		if !oldID.Valid {
			response.Fail(w, "该家属账号尚未关联老人，请联系管理员")
			return
		}
		h.renewSession(w, r, session.Principal{
			Role: auth.RoleFamily, ID: fid, Phone: fPhone.String,
			OldID: oldID.Int64, DisplayName: fName.String,
		})
		response.OK(w, map[string]any{
			"family": map[string]any{
				"oldId": oldID.Int64, "name": oldName.String,
				"familyId": fid, "familyName": fName.String, "familyRelation": fRel.String,
				"familyPhone": fPhone.String, "familySex": fSex.String,
				"familyAddress": fAddr.String, "familyTouPic": fTouPic.String,
			},
		})
		return
	}

	response.Fail(w, "手机号或密码不正确")
}

// renewSession 登录建会话：销毁请求 cookie 带来的旧会话（防 session fixation），建新会话并写 cookie。
func (h *Handler) renewSession(w http.ResponseWriter, r *http.Request, p session.Principal) {
	if c, err := r.Cookie(session.CookieName); err == nil && c.Value != "" {
		h.Sessions.Destroy(c.Value)
	}
	sess := h.Sessions.Create(p)
	session.SetCookie(w, sess.ID)
}

// logout 复刻 POST /restful/userLogin/logout：需登录（中间件鉴权），销毁会话并清 cookie。
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	sess := session.FromContext(r)
	if sess == nil {
		sess = h.Sessions.FromRequest(r)
	}
	if sess != nil {
		h.Sessions.Destroy(sess.ID)
	}
	session.ClearCookie(w)
	response.OK(w, nil)
}

// setup 复刻 POST /restful/setup：公开 + 桌面 token；一次性初始化管理员账号。
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
	if err != nil {
		response.Error500(w, ref())
		return
	}
	defer tx.Rollback(ctx)

	// 行锁串行化，防止并发重复初始化。
	var metaID int64
	if err := tx.QueryRow(ctx, "SELECT id FROM app_metadata WHERE id=1 FOR UPDATE").Scan(&metaID); err != nil {
		response.Error500(w, ref())
		return
	}
	var count int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		response.Error500(w, ref())
		return
	}
	if count != 0 {
		response.JSON(w, http.StatusConflict, map[string]any{"returnCode": "0000", "message": "管理员已创建"})
		return
	}
	if !auth.ValidPhone(phone) || strings.TrimSpace(name) == "" ||
		utf8.RuneCountInString(name) > 10 || !auth.ValidNewPassword(password) {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": "请检查姓名、手机号和密码"})
		return
	}
	hashed, err := auth.Hash(password)
	if err != nil {
		response.Error500(w, ref())
		return
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO users(user_name,user_phone,password,role) VALUES($1,$2,$3,'1')",
		name, phone, hashed); err != nil {
		response.Error500(w, ref())
		return
	}
	if err := tx.Commit(ctx); err != nil {
		response.Error500(w, ref())
		return
	}
	response.OK(w, nil)
}

// desktopExit 复刻 POST /restful/desktop/exit：公开 + 桌面 token；通过后延迟退出进程。
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
