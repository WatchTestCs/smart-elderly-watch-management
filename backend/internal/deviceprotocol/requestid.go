package deviceprotocol

var requestIDByte = func() [256]byte {
	var valid [256]byte
	for c := 'a'; c <= 'z'; c++ {
		valid[c] = 1
		valid[c-32] = 1
	}
	for c := '0'; c <= '9'; c++ {
		valid[c] = 1
	}
	for _, c := range "_.:-" {
		valid[c] = 1
	}
	return valid
}()

// ValidRequestID enforces the common SOS and durable telemetry event ID format.
func ValidRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		if requestIDByte[id[i]] == 0 {
			return false
		}
	}
	return true
}
