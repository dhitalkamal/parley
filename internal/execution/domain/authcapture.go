package execution

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// ExtractJSONField walks dotPath (e.g. "data.tokens.access") through body's
// parsed JSON and returns the leaf value stringified - this is the
// structured equivalent of a hand-written
// pm.environment.set("token", pm.response.json().data.tokens.access), used
// by AuthCapture (see send.go's applyAuthCapture) so a user never has to
// write that script by hand for the common "extract a field, save it"
// case. ok is false for an empty path, invalid JSON, a missing key at any
// step, or a path that continues past a leaf (non-object) value.
func ExtractJSONField(body []byte, dotPath string) (string, bool) {
	if dotPath == "" {
		return "", false
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", false
	}
	cur := parsed
	for _, key := range strings.Split(dotPath, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		v, ok := m[key]
		if !ok {
			return "", false
		}
		cur = v
	}
	return stringifyJSONValue(cur)
}

func stringifyJSONValue(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	default:
		return "", false
	}
}

// ComputeExpiresAt adds secondsStr (a captured "expires in N seconds"
// field) to now and returns the resulting absolute Unix timestamp as a
// string, ready to store as a plain environment variable - IsExpired below
// is its reader.
func ComputeExpiresAt(now time.Time, secondsStr string) (string, bool) {
	seconds, err := strconv.ParseFloat(secondsStr, 64)
	if err != nil {
		return "", false
	}
	expiresAt := now.Add(time.Duration(seconds * float64(time.Second)))
	return strconv.FormatInt(expiresAt.Unix(), 10), true
}

// IsExpired reports whether now is at or past expiresAtStr (a Unix
// timestamp string previously written by ComputeExpiresAt). An empty or
// unparsable value is treated as "not expired" - RefreshConfig's proactive
// check has nothing to act on without a real recorded expiry, and should
// fail open (send as normal) rather than refresh on every single send.
func IsExpired(expiresAtStr string, now time.Time) bool {
	expiresAt, err := strconv.ParseInt(expiresAtStr, 10, 64)
	if err != nil {
		return false
	}
	return !now.Before(time.Unix(expiresAt, 0))
}
