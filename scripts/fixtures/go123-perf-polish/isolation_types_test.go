package health

import (
	"crypto/sha256"
	"encoding/hex"
)

type Owner struct {
	OldID    int64
	DeviceID *int64
	Epoch    int64
}

func Digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
