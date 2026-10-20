package session_test

import (
	"careold-go/internal/db/dbtest"
	"careold-go/internal/securestore"
	"careold-go/internal/session"
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestDatabaseSessionsShareRevokeAndFactorGeneration(t *testing.T) {
	pool := dbtest.MySQL(t)
	box, _ := securestore.New(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	a, b := session.NewStore(time.Hour), session.NewStore(time.Hour)
	a.UseDatabase(pool.SQL(), box)
	b.UseDatabase(pool.SQL(), box)
	s := a.Create(session.Principal{Role: "ADMIN", ID: 1, Phone: "13800000001"})
	if s == nil {
		t.Fatal("create")
	}
	if got := b.Get(s.ID); got == nil || got.CSRFToken != s.CSRFToken {
		t.Fatal("replica cannot read session")
	}
	var payload, hash string
	if err := pool.QueryRow(context.Background(), "SELECT payload,id_hash FROM web_session LIMIT 1").Scan(&payload, &hash); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "13800000001") || strings.Contains(payload, s.CSRFToken) || hash == s.ID {
		t.Fatal("session secrets are plaintext")
	}
	if !a.Destroy(s.ID) || b.Get(s.ID) != nil {
		t.Fatal("cross-replica revocation failed")
	}
	old := a.Create(session.Principal{Role: "ADMIN", ID: 1})
	_, err := pool.Exec(context.Background(), "INSERT INTO auth_factor(role,actor_id,secret,generation,enabled,pending_until,recovery_hashes) VALUES('ADMIN',1,'encrypted','new-generation',true,now(),'[]')")
	if err != nil {
		t.Fatal(err)
	}
	if b.Get(old.ID) != nil {
		t.Fatal("session creation racing factor enable bypassed MFA")
	}
	current := a.Create(session.Principal{Role: "ADMIN", ID: 1, FactorGeneration: "new-generation"})
	if current == nil || b.Get(current.ID) == nil {
		t.Fatal("verified generation rejected")
	}
	if err = a.Revoke("ADMIN", 1); err != nil || b.Get(current.ID) != nil {
		t.Fatal("principal revocation failed", err)
	}
	if _, err = pool.Exec(context.Background(), "DELETE FROM auth_factor WHERE actor_id=1"); err != nil {
		t.Fatal(err)
	}
	late := a.Create(session.Principal{Role: "ADMIN", ID: 1, FactorGeneration: "new-generation"})
	if late == nil || b.Get(late.ID) != nil {
		t.Fatal("session creation racing factor disable retained the old generation")
	}
	pool.Close()
	if a.Create(session.Principal{}) != nil || b.Get(old.ID) != nil || a.Destroy(old.ID) {
		t.Fatal("storage failure did not fail closed")
	}
}
