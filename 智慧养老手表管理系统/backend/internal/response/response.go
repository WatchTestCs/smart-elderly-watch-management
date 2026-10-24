// Package response 复刻 Java 端的响应包络：
// 成功 {"returnCode":"1111", ...}，业务失败 {"returnCode":"0000","message":...}。
// 注意：部分纯查询接口按源码行为不带 returnCode，用 JSON 直接写。
package response

import (
	"encoding/json"
	"net/http"
)

const (
	CodeOK   = "1111"
	CodeFail = "0000"
)

// contentTypeJSON 预分配的响应头值："Content-Type" 已是规范形式，
// 直接写入 Header map 可跳过 Set 的键规范化与单元素切片分配。
// net/http 只读取响应头值切片，包级共享安全。
var contentTypeJSON = []string{"application/json;charset=UTF-8"}

// newLine 单独一次 Write，避免 append 触发 json.Marshal 结果扩容拷贝
//（与旧 json.Encoder 在内部缓冲追加 '\n' 后单次写出字节流一致）。
var newLine = []byte{'\n'}

// okEmptyEnvelope 空成功包络（22 处 OK(w, nil)/OK(w, map{} 调用形状）。
var okEmptyEnvelope = []byte("{\"returnCode\":\"" + CodeOK + "\"}\n")

// err500Prefix/err500Suffix 按 json.Marshal 对 map 键排序后的固定字段序：
// message < reference < returnCode。
const (
	err500Message = "操作未完成，请重试；如仍失败，请提供此编号以便排查"
	err500Prefix  = `{"message":"` + err500Message + `","reference":"`
	err500Suffix  = `","returnCode":"` + CodeFail + `"}`
)

// setJSONHeader 写响应头与状态码（各出口统一入口，避免重复 WriteHeader）。
func setJSONHeader(w http.ResponseWriter, status int) {
	w.Header()["Content-Type"] = contentTypeJSON
	w.WriteHeader(status)
}

// writeJSONBody 编码 v 并补换行；Marshal 失败时 body 为空，
// 与旧实现忽略 Encoder.Encode 错误的行为一致。
func writeJSONBody(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = w.Write(data)
	_, _ = w.Write(newLine)
}

// JSON 直接写任意 JSON（用于不带 returnCode 的查询接口）。
func JSON(w http.ResponseWriter, status int, v any) {
	setJSONHeader(w, status)
	writeJSONBody(w, v)
}

// OK 写成功包络，可附带业务字段。
func OK(w http.ResponseWriter, fields map[string]any) {
	if len(fields) == 0 { // nil 或空 map：常量包络，免 map 构造与编码
		setJSONHeader(w, http.StatusOK)
		_, _ = w.Write(okEmptyEnvelope)
		return
	}
	fields["returnCode"] = CodeOK
	setJSONHeader(w, http.StatusOK)
	writeJSONBody(w, fields)
}

// Fail 写业务失败包络（HTTP 200，returnCode=0000）。
// 手工拼装 {"message":<msg>,"returnCode":"0000"}：message 值经
// json.Marshal 转义，与旧 map 编码逐字节一致（键序为排序结果）。
func Fail(w http.ResponseWriter, message string) {
	setJSONHeader(w, http.StatusOK)
	msgJSON, _ := json.Marshal(message)
	var buf [192]byte
	b := append(buf[:0], `{"message":`...)
	b = append(b, msgJSON...)
	b = append(b, `,"returnCode":"`...)
	b = append(b, CodeFail...)
	b = append(b, '"', '}', '\n')
	_, _ = w.Write(b)
}

// Error500 映射 DataAccessException/IllegalStateException：500 + reference UUID。
// 调用方均传 uuid.NewString()（hex+连字符，无需 JSON 转义），直接拼装固定
// 字段序包络，免 map 构造与反射编码；非常规 reference 回退旧 map 编码路径。
func Error500(w http.ResponseWriter, reference string) {
	setJSONHeader(w, http.StatusInternalServerError)
	if isUUID(reference) {
		var buf [192]byte
		b := append(buf[:0], err500Prefix...)
		b = append(b, reference...)
		b = append(b, err500Suffix...)
		b = append(b, '\n')
		_, _ = w.Write(b)
		return
	}
	writeJSONBody(w, map[string]any{
		"returnCode": CodeFail,
		"message":    err500Message,
		"reference":  reference,
	})
}

// isUUID 报告 s 是否为 8-4-4-4-12 的十六进制 UUID 形式（uuid.NewString 输出）。
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if s[i] != '-' {
				return false
			}
			continue
		}
		c := s[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F') {
			return false
		}
	}
	return true
}
