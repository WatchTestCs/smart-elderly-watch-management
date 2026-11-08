package handlers

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestContinuousElderGuardianMustExist(t *testing.T) {
	h := extraAuditHandler(t)
	mustExec(t, h.DB, "INSERT INTO guardian(guardian_id,guardian_name) VALUES(1,'甲'),(2,'乙')")
	body := elderPayload()
	body["guardianId"] = 999
	if out := apiFailure(t, 400, h.createResident, body); out["returnCode"] == "1111" {
		t.Error("elder creation accepted a nonexistent guardian")
	}
	var count int
	if err := h.DB.QueryRow(context.Background(), "SELECT count(*) FROM resident").Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid guardian wrote an elder: count=%d err=%v", count, err)
	}
	body["guardianId"] = 1
	if out := api(t, h.createResident, body); out["returnCode"] != "1111" {
		t.Fatal(out)
	}
	var residentID int64
	if err := h.DB.QueryRow(context.Background(), "SELECT resident_id FROM resident").Scan(&residentID); err != nil {
		t.Fatal(err)
	}
	body["residentId"], body["guardianId"], body["name"] = residentID, 999, "不应写入"
	if out := apiFailure(t, 400, h.updateResident, body); out["returnCode"] == "1111" {
		t.Error("elder edit accepted a nonexistent guardian")
	}
	var guardian int64
	var name string
	if err := h.DB.QueryRow(context.Background(), "SELECT guardian_id,name FROM resident WHERE resident_id=$1", residentID).Scan(&guardian, &name); err != nil || guardian != 1 || name != "长者" {
		t.Fatalf("rejected edit changed the profile: guardian=%d name=%s err=%v", guardian, name, err)
	}
	body["guardianId"], body["name"] = 2, "长者"
	if out := api(t, h.updateResident, body); out["returnCode"] != "1111" {
		t.Fatal(out)
	}
	detail := api(t, h.getResidentDetails, map[string]any{"name": ""})["residentDetails"].([]any)[0].(map[string]any)
	if detail["guardianId"] != float64(2) || detail["guardianName"] != "乙" {
		t.Fatalf("valid guardian reassignment lost: %v", detail)
	}
}

func TestContinuousGuardianDeleteKeepsAssignedProfiles(t *testing.T) {
	h := extraAuditHandler(t)
	mustExec(t, h.DB, "INSERT INTO guardian(guardian_id,guardian_name) VALUES(1,'甲'),(2,'乙')")
	mustExec(t, h.DB, "INSERT INTO resident(resident_id,name,guardian_id) VALUES(1,'长者',1)")
	if out := apiFailure(t, 400, h.deleteGuardian, map[string]any{"guardianId": 1}); out["returnCode"] == "1111" {
		t.Error("deleted a guardian still assigned to an elder")
	}
	var count int
	if err := h.DB.QueryRow(context.Background(), "SELECT count(*) FROM resident o JOIN guardian c ON c.guardian_id=o.guardian_id WHERE o.resident_id=1").Scan(&count); err != nil || count != 1 {
		t.Fatalf("guardian reference became dangling: count=%d err=%v", count, err)
	}
	if out := api(t, h.deleteGuardian, map[string]any{"guardianId": 2}); out["returnCode"] != "1111" {
		t.Fatalf("unassigned guardian could not be deleted: %v", out)
	}
	if out := api(t, h.deleteResident, map[string]any{"residentId": 1}); out["returnCode"] != "1111" {
		t.Fatal(out)
	}
	if out := api(t, h.deleteGuardian, map[string]any{"guardianId": 1}); out["returnCode"] != "1111" {
		t.Fatalf("guardian could not be deleted after elder removal: %v", out)
	}
}

func TestContinuousGuardianAgeBoundsDoNotWriteInvalidProfiles(t *testing.T) {
	h := extraAuditHandler(t)
	mustExec(t, h.DB, "INSERT INTO guardian(guardian_id,guardian_name,guardian_age) VALUES(1,'原姓名',40)")
	body := map[string]any{"guardianId": 1, "guardianName": "不应写入", "guardianSex": "女", "guardianPhone": "13800000001", "guardianAddress": "测试"}
	for _, age := range []any{-1, 151, 1.5, "invalid"} {
		body["guardianAge"] = age
		for _, handler := range []struct {
			name string
			call func() int
		}{{"add", func() int { return callAPI(h.createGuardian, body).Code }}, {"edit", func() int { return callAPI(h.updateGuardian, body).Code }}} {
			if code := handler.call(); code != 400 {
				t.Errorf("%s age=%v: HTTP %d, want 400", handler.name, age, code)
			}
		}
	}
	var count, age int
	var name string
	if err := h.DB.QueryRow(context.Background(), "SELECT count(*) FROM guardian").Scan(&count); err != nil || count != 1 {
		t.Fatalf("invalid ages created guardians: count=%d err=%v", count, err)
	}
	if err := h.DB.QueryRow(context.Background(), "SELECT guardian_age,guardian_name FROM guardian WHERE guardian_id=1").Scan(&age, &name); err != nil || age != 40 || name != "原姓名" {
		t.Fatalf("invalid ages overwrote a guardian: age=%d name=%s err=%v", age, name, err)
	}
	for _, valid := range []any{0, "150"} {
		body["guardianAge"] = valid
		if out := api(t, h.updateGuardian, body); out["returnCode"] != "1111" {
			t.Fatalf("valid age %v rejected: %v", valid, out)
		}
	}
}

func TestContinuousGuardianCreateAndDeleteKeepReferencesValid(t *testing.T) {
	h := extraAuditHandler(t)
	mustExec(t, h.DB, "INSERT INTO guardian(guardian_id,guardian_name) VALUES(1,'监护人')")
	start := make(chan struct{})
	created := make(chan *httptest.ResponseRecorder, 1)
	deleted := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		<-start
		created <- callAPI(h.createResident, elderPayload())
	}()
	go func() {
		<-start
		deleted <- callAPI(h.deleteGuardian, map[string]any{"guardianId": 1})
	}()
	close(start)
	createResponse, deleteResponse := <-created, <-deleted
	if (createResponse.Code != 200 && createResponse.Code != 400) || (deleteResponse.Code != 200 && deleteResponse.Code != 400) {
		t.Fatalf("concurrent requests must finish without server errors: create=%d %s delete=%d %s", createResponse.Code, createResponse.Body.String(), deleteResponse.Code, deleteResponse.Body.String())
	}
	createResult, deleteResult := architectureBody(t, createResponse), architectureBody(t, deleteResponse)
	createOK, deleteOK := createResult["returnCode"] == "1111", deleteResult["returnCode"] == "1111"
	if createOK == deleteOK {
		t.Fatalf("assignment and parent deletion must have exactly one winner: create=%v delete=%v", createResult, deleteResult)
	}
	var dangling int
	if err := h.DB.QueryRow(context.Background(), "SELECT count(*) FROM resident o LEFT JOIN guardian c ON c.guardian_id=o.guardian_id WHERE o.guardian_id IS NOT NULL AND c.guardian_id IS NULL").Scan(&dangling); err != nil || dangling != 0 {
		t.Fatalf("concurrent requests left dangling references: count=%d err=%v", dangling, err)
	}
}
