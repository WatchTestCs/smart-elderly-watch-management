package response

import (
	"encoding/json"
	"errors"
	"net/http"
)

// BaseResponse preserves the existing flat JSON response fields.
type BaseResponse struct {
	ReturnCode string `json:"returnCode"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
}

// Envelope is for endpoints whose public contract explicitly uses a data field.
// Existing flat endpoints must use WriteStruct with their own DTO instead.
type Envelope[T any] struct {
	BaseResponse
	Data T `json:"data,omitempty"`
}

// AppError makes the HTTP status and domain error code explicit.
type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Err        error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func BadRequest(msg string, err error) *AppError {
	return &AppError{HTTPStatus: http.StatusBadRequest, Message: msg, Err: err}
}
func Unauthorized(msg string, err error) *AppError {
	return &AppError{HTTPStatus: http.StatusUnauthorized, Message: msg, Err: err}
}
func Forbidden(msg string, err error) *AppError {
	return &AppError{HTTPStatus: http.StatusForbidden, Message: msg, Err: err}
}
func NotFound(msg string, err error) *AppError {
	return &AppError{HTTPStatus: http.StatusNotFound, Message: msg, Err: err}
}
func Conflict(msg string, err error) *AppError {
	return &AppError{HTTPStatus: http.StatusConflict, Message: msg, Err: err}
}
func Internal(msg string, err error) *AppError {
	return &AppError{HTTPStatus: http.StatusInternalServerError, Message: msg, Err: err}
}

// WriteStruct emits the supplied public DTO and preserves its exact status.
func WriteStruct[T any](w http.ResponseWriter, status int, payload T) {
	writeTypedJSON(w, status, payload)
}

// WriteOK is only for endpoints with an explicit {returnCode,data} contract.
func WriteOK[T any](w http.ResponseWriter, data T) {
	writeTypedJSON(w, http.StatusOK, Envelope[T]{BaseResponse: BaseResponse{ReturnCode: CodeOK}, Data: data})
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		writeTypedJSON(w, http.StatusInternalServerError, BaseResponse{ReturnCode: CodeFail, Code: "INTERNAL_ERROR", Message: "internal server error"})
		return
	}
	status := appErr.HTTPStatus
	if status < 400 || status > 599 {
		status = http.StatusInternalServerError
	}
	defaultCode, defaultMessage := errorDefaults(status)
	code := appErr.Code
	if code == "" {
		code = defaultCode
	}
	message := appErr.Message
	if message == "" {
		message = defaultMessage
	}
	writeTypedJSON(w, status, BaseResponse{ReturnCode: CodeFail, Code: code, Message: message})
}

func writeTypedJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
