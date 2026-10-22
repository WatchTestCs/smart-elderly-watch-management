package health

import (
	"encoding/json"
	"math"
	"strconv"
	"time"
)

// marshalSampleFast matches encoding/json's struct-field ordering and bytes
// for the common, fully normalized, scalar health sample. Uncommon or
// untrusted representations always use encoding/json for exact compatibility.
func marshalSampleFast(s Sample) ([]byte, error) {
	if s.Value == nil || !finiteSample(*s.Value) || s.SourceVersion != "" || s.GroupID != "" || s.Stage != "" || s.TimeBasis != "" || len(s.Metadata) != 0 || s.Deleted ||
		!safeSampleString(s.RecordID) || !safeSampleString(s.Source) || !safeSampleString(s.SourceDevice) ||
		!safeSampleString(s.Metric) || !safeSampleString(s.Unit) || !safeSampleString(s.Semantics) ||
		!safeSampleTime(s.Start) || !safeSampleTime(s.End) {
		return json.Marshal(s)
	}
	// Lengths of strings vary widely across devices. Allocate only one output
	// buffer; a bounded reserve covers common timestamps and measurements.
	// Append grows safely for unusually long RFC3339Nano/float representations.
	b := make([]byte, 0, 208+len(s.RecordID)+len(s.Source)+len(s.SourceDevice)+len(s.Metric)+len(s.Unit)+len(s.Semantics))
	b = append(b, `{"recordId":"`...)
	b = append(b, s.RecordID...)
	b = append(b, `","source":"`...)
	b = append(b, s.Source...)
	b = append(b, `","sourceDevice":"`...)
	b = append(b, s.SourceDevice...)
	b = append(b, `","metric":"`...)
	b = append(b, s.Metric...)
	b = append(b, `","unit":"`...)
	b = append(b, s.Unit...)
	b = append(b, `","value":`...)
	b = appendFloatSample(b, *s.Value)
	b = append(b, `,"start":"`...)
	b = s.Start.AppendFormat(b, time.RFC3339Nano)
	b = append(b, `","end":"`...)
	b = s.End.AppendFormat(b, time.RFC3339Nano)
	b = append(b, `","semantics":"`...)
	b = append(b, s.Semantics...)
	b = append(b, `","version":`...)
	b = strconv.AppendInt(b, s.Version, 10)
	return append(b, '}'), nil
}

func safeSampleString(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c >= 0x80 || c == '"' || c == '\\' || c == '<' || c == '>' || c == '&' {
			return false
		}
	}
	return true
}
func safeSampleTime(t time.Time) bool {
	year := t.Year()
	_, offset := t.Zone()
	return year >= 0 && year <= 9999 && offset%60 == 0 && offset > -24*3600 && offset < 24*3600
}
func finiteSample(f float64) bool { return !math.IsNaN(f) && !math.IsInf(f, 0) }
func appendFloatSample(b []byte, v float64) []byte {
	if v != 0 && v > -1e15 && v < 1e15 && v == math.Trunc(v) {
		return strconv.AppendInt(b, int64(v), 10)
	}
	format := byte('f')
	abs := math.Abs(v)
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	b = strconv.AppendFloat(b, v, format, -1, 64)
	if format == 'e' {
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}
	return b
}
