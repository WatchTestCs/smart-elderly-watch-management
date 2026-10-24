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
		!safeSampleString(s.Metric) || !safeSampleString(s.Unit) || !safeSampleString(s.Semantics) {
		return json.Marshal(s)
	}
	start, startOK := civilSample(s.Start)
	end, endOK := civilSample(s.End)
	if !startOK || !endOK {
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
	b = appendSampleClock(b, start)
	b = append(b, `","end":"`...)
	b = appendSampleClock(b, end)
	b = append(b, `","semantics":"`...)
	b = append(b, s.Semantics...)
	b = append(b, `","version":`...)
	b = strconv.AppendInt(b, s.Version, 10)
	return append(b, '}'), nil
}

// unsafeSampleByte transliterates the original per-byte guard
// (c < 0x20 || c >= 0x80 || c == '"' || c == '\\' || c == '<' || c == '>' || c == '&')
// into one table lookup shared by every fast-path scanner.
var unsafeSampleByte = func() (t [256]bool) {
	for c := range 256 {
		t[c] = c < 0x20 || c >= 0x80 || c == '"' || c == '\\' || c == '<' || c == '>' || c == '&'
	}
	return
}()

func safeSampleString(s string) bool {
	for i := 0; i < len(s); i++ {
		if unsafeSampleByte[s[i]] {
			return false
		}
	}
	return true
}

// sampleClock is the local wall-clock decomposition of one timestamp.
type sampleClock struct {
	year, month, day, hour, minute, second, nanos, offset int
}

// civilSample resolves the wall-clock fields with a single zone lookup and
// reports whether the timestamp fits the previous safeSampleTime guard
// (proleptic-Gregorian year 0..9999, whole-minute offset within one day).
func civilSample(t time.Time) (sampleClock, bool) {
	_, offset := t.Zone()
	if offset%60 != 0 || offset <= -24*3600 || offset >= 24*3600 {
		return sampleClock{}, false
	}
	u := t.Unix() + int64(offset)
	days, sec := u/86400, u%86400
	if sec < 0 {
		sec += 86400
		days--
	}
	// days-to-civil conversion (proleptic Gregorian, Howard Hinnant's era form).
	z := days + 719468
	era := z / 146097
	if z%146097 < 0 {
		era--
	}
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	year := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	day := doy - (153*mp+2)/5 + 1
	month := mp + 3
	if mp >= 10 {
		month = mp - 9
		year++
	}
	if year < 0 || year > 9999 {
		return sampleClock{}, false
	}
	return sampleClock{
		year:   int(year),
		month:  int(month),
		day:    int(day),
		hour:   int(sec / 3600),
		minute: int(sec % 3600 / 60),
		second: int(sec % 60),
		nanos:  t.Nanosecond(),
		offset: offset,
	}, true
}

// appendSampleClock writes exactly the bytes of t.AppendFormat(b, time.RFC3339Nano)
// for every timestamp accepted by civilSample.
func appendSampleClock(b []byte, c sampleClock) []byte {
	b = appendDigits(b, c.year, 4)
	b = append(b, '-')
	b = appendDigits(b, c.month, 2)
	b = append(b, '-')
	b = appendDigits(b, c.day, 2)
	b = append(b, 'T')
	b = appendDigits(b, c.hour, 2)
	b = append(b, ':')
	b = appendDigits(b, c.minute, 2)
	b = append(b, ':')
	b = appendDigits(b, c.second, 2)
	if c.nanos != 0 {
		b = append(b, '.')
		b = appendDigits(b, c.nanos, 9)
		for b[len(b)-1] == '0' {
			b = b[:len(b)-1]
		}
	}
	if c.offset == 0 {
		return append(b, 'Z')
	}
	offset := c.offset
	if offset < 0 {
		b = append(b, '-')
		offset = -offset
	} else {
		b = append(b, '+')
	}
	b = appendDigits(b, offset/3600, 2)
	b = append(b, ':')
	return appendDigits(b, offset/60%60, 2)
}

// digitPairs holds the two-decimal-digit forms 00..99 for fixed-width appends.
var digitPairs = func() (t [200]byte) {
	for i := range 100 {
		t[2*i] = byte('0' + i/10)
		t[2*i+1] = byte('0' + i%10)
	}
	return
}()

func appendDigits(b []byte, v, n int) []byte {
	switch n {
	case 2: // v <= 99
		return append(b, digitPairs[2*v:2*v+2]...)
	case 4: // v <= 9999
		b = append(b, digitPairs[2*(v/100):2*(v/100)+2]...)
		return append(b, digitPairs[2*(v%100):2*(v%100)+2]...)
	case 9: // v <= 999999999
		b = append(b, byte('0'+v/100000000))
		v %= 100000000
		b = append(b, digitPairs[2*(v/1000000):2*(v/1000000)+2]...)
		b = append(b, digitPairs[2*(v/10000%100):2*(v/10000%100)+2]...)
		b = append(b, digitPairs[2*(v/100%100):2*(v/100%100)+2]...)
		return append(b, digitPairs[2*(v%100):2*(v%100)+2]...)
	}
	var tmp [9]byte
	for i := n - 1; i >= 0; i-- {
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(b, tmp[:n]...)
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
