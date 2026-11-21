package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteStructPreservesFlatContractAndStatus(t *testing.T) {
	type reply struct {
		Data string `json:"data"`
		BaseResponse
	}
	w := httptest.NewRecorder()
	WriteStruct(w, http.StatusAccepted, reply{Data: "ok", BaseResponse: BaseResponse{ReturnCode: CodeOK}})
	if w.Code != http.StatusAccepted || w.Body.String() != "{\"data\":\"ok\",\"returnCode\":\"1111\"}\n" {
		t.Fatalf("flat JSON contract changed: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWriteErrorPreservesDomainCode(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"unauthorized", Unauthorized("please login", nil), 401, "AUTH_REQUIRED"},
		{"conflict", &AppError{HTTPStatus: 409, Code: "HANDOVER_PENDING", Message: "pending"}, 409, "HANDOVER_PENDING"},
		{"untyped", errors.New("private SQL details"), 500, "INTERNAL_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tc.err)
			var got struct {
				ReturnCode string `json:"returnCode"`
				Code       string `json:"code"`
				Message    string `json:"message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.status || got.ReturnCode != CodeFail || got.Code != tc.code || got.Message == "" {
				t.Fatalf("unexpected error contract: %d %+v", w.Code, got)
			}
		})
	}
}
