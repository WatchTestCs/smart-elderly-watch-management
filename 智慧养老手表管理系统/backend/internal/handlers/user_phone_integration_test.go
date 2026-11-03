package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"suining-care/internal/auth"
	"suining-care/internal/securestore"
)

func adminPhoneTestBox(t testing.TB) *securestore.Box {
	t.Helper()
	box, err := securestore.NewKeyring("A", map[string]string{"A": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))}, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestProtectedAdministratorHTTP(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(t, pool)
	h.Cfg.DesktopToken = "synthetic-admin-phone-bootstrap"
	phone, password := "13800000001", "TestPassword123!"
	setup := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest("POST", "/api/v1/setup", strings.NewReader(`{"name":"synthetic","phone":"13800000001","password":"TestPassword123!"}`))
		request.Header.Set("X-Suining-Desktop", h.Cfg.DesktopToken)
		response := httptest.NewRecorder()
		h.setup(response, request)
		return response
	}
	if response := setup(); response.Code != 500 {
		t.Fatal("missing encryption key did not reject setup", response.Code)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&count); err != nil || count != 0 {
		t.Fatal("failed setup retained a partial administrator", err)
	}
	h.Box = adminPhoneTestBox(t)
	if response := setup(); response.Code != 200 {
		t.Fatal("protected setup failed", response.Code, response.Body.String())
	}
	var id int64
	var legacy, cipher, lookup *string
	var protected bool
	if err := pool.QueryRow(context.Background(), "SELECT user_id,user_phone,user_phone_cipher,user_phone_lookup,user_phone_protected FROM users").Scan(&id, &legacy, &cipher, &lookup, &protected); err != nil || legacy != nil || !protected || cipher == nil || lookup == nil || strings.Contains(*cipher, phone) {
		t.Fatal("setup persisted the administrator phone in plaintext", err)
	}
	assertAdminLogin := func() {
		t.Helper()
		response := callAPI(h.login, map[string]any{"phone": phone, "password": password})
		cookies := response.Result().Cookies()
		if response.Code != 200 || !strings.Contains(response.Body.String(), `"authenticated":true`) || len(cookies) != 1 {
			t.Fatal("protected administrator login changed its HTTP contract", response.Code)
		}
		current := h.Sessions.Get(cookies[0].Value)
		if current == nil || current.Principal.Role != auth.RoleAdmin || current.Principal.ID != id || current.Principal.Phone != phone {
			t.Fatal("protected login did not persist the correct administrator principal")
		}
	}
	assertAdminLogin()
	if response := callAPI(h.userSelect, map[string]any{"userPhone": phone}); response.Code != 200 || !strings.Contains(response.Body.String(), `"userPhone":"13800000001"`) || strings.Contains(response.Body.String(), securestore.Prefix) || strings.Contains(response.Body.String(), *lookup) {
		t.Fatal("account lookup did not return only the authorized plaintext DTO", response.Code, response.Body.String())
	}
	// Existing installations can contain both roles with one phone. Preserve
	// administrator-first login and refuse creating/re-activating more conflicts.
	hash, err := auth.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, pool, "INSERT INTO family(family_id,family_name,family_phone,family_password) VALUES(1,'legacy',$1,$2)", phone, hash)
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name,phone,family_id) VALUES(1,'synthetic','13800000011',1),(2,'unbound','13800000012',NULL)")
	assertAdminLogin()
	family := map[string]any{"familyId": 1, "familyName": "synthetic", "familyRelation": "child", "familySex": "男", "familyPhone": phone, "oldPhone": "13800000012", "familyAddress": "synthetic"}
	for _, handler := range []http.HandlerFunc{h.familyAdd, h.familyUpdate} {
		if response := callAPI(handler, family); response.Code != 400 || !strings.Contains(response.Body.String(), "该手机号已被使用") {
			t.Fatal("encrypted administrator did not block duplicate family write", response.Code, response.Body.String())
		}
	}
	// Remove the family phone collision to prove duplicate detection came from
	// the protected administrator index, not from the remaining legacy column.
	mustExec(t, pool, "UPDATE family SET family_phone='13800000002',family_password=NULL WHERE family_id=1")
	for _, handler := range []http.HandlerFunc{h.familyAdd, h.familyUpdate} {
		if response := callAPI(handler, family); response.Code != 400 || !strings.Contains(response.Body.String(), "该手机号已被使用") {
			t.Fatal("encrypted-only administrator duplicate was missed", response.Code, response.Body.String())
		}
	}
	mustExec(t, pool, "INSERT INTO invite_code(code,old_id,used) VALUES('PHONEBOUND',1,0),('PHONEOPEN',2,0)")
	registration := map[string]any{"familyName": "synthetic", "familyPhone": phone, "familyPassword": password, "familyRelation": "child", "familySex": "男", "familyAddress": "synthetic"}
	for _, code := range []string{"PHONEBOUND", "PHONEOPEN"} {
		registration["code"] = code
		if response := callAPI(h.inviteJoin, registration); response.Code != 400 {
			t.Fatal("invitation registered a conflicting encrypted administrator phone", response.Code, response.Body.String())
		}
	}
	var used int
	if err := pool.QueryRow(context.Background(), "SELECT SUM(used) FROM invite_code").Scan(&used); err != nil || used != 0 {
		t.Fatal("duplicate invitation consumed a code", err)
	}
	// A corrupt encrypted row never falls back to a plaintext family account,
	// even if that account would otherwise accept the password.
	mustExec(t, pool, "UPDATE family SET family_phone=$1,family_password=$2 WHERE family_id=1", phone, hash)
	mustExec(t, pool, "UPDATE users SET user_phone_cipher='sne1:invalid' WHERE user_id=$1", id)
	if response := callAPI(h.login, map[string]any{"phone": phone, "password": password}); response.Code != 500 || strings.Contains(response.Body.String(), "sne1:") {
		t.Fatal("invalid administrator envelope fell back to another role or leaked data", response.Code)
	}
	mustExec(t, pool, "UPDATE users SET user_phone_cipher=$1 WHERE user_id=$2", cipher, id)
	correctBox := h.Box
	wrongLookup, err := securestore.NewKeyring("A", map[string]string{"A": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))}, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{4}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	h.Box = wrongLookup
	if response := callAPI(h.login, map[string]any{"phone": phone, "password": password}); response.Code != 500 {
		t.Fatal("wrong lookup key made an administrator disappear or permitted family fallback", response.Code)
	}
	if response := callAPI(h.familyAdd, family); response.Code != 500 {
		t.Fatal("wrong lookup key bypassed cross-role duplicate protection", response.Code)
	}
	h.Box = correctBox
	var verifier string
	if err := pool.QueryRow(context.Background(), "SELECT lookup_verifier FROM account_phone_key_state WHERE singleton=1").Scan(&verifier); err != nil {
		t.Fatal(err)
	}
	mustExec(t, pool, "DELETE FROM account_phone_key_state WHERE singleton=1")
	if response := callAPI(h.login, map[string]any{"phone": phone, "password": password}); response.Code != 500 {
		t.Fatal("missing lookup binding was silently rebuilt from a login request", response.Code)
	}
	mustExec(t, pool, "INSERT INTO account_phone_key_state(singleton,lookup_verifier) VALUES(1,$1)", verifier)
	h.Box = nil
	if response := callAPI(h.login, map[string]any{"phone": phone, "password": password}); response.Code != 500 {
		t.Fatal("missing key silently omitted protected administrators", response.Code)
	}
	if response := callAPI(h.familyAdd, family); response.Code != 500 {
		t.Fatal("missing key bypassed cross-role duplicate protection", response.Code)
	}
}
