package tui

import (
	"regexp"
	"strings"
)

// secretKeyPattern matches a key name that looks like it holds a secret,
// case-insensitively, as a substring so keys like "apiToken" or "auth_token"
// still match. The optional [_-]? separators cover the common casings
// (apiKey, api_key, api-key, x-api-key). Covers passwords, tokens, generic
// secrets, authorization, api/access keys, oauth client ids, cookies (incl.
// Set-Cookie), credentials, and private keys - the on-screen masking control
// has to catch these so they are not exposed during a screen-share.
var secretKeyPattern = regexp.MustCompile(`(?i)password|passwd|token|secret|authorization|api[_-]?key|access[_-]?key|client[_-]?id|cookie|credential|private[_-]?key`)

func looksLikeSecretKey(key string) bool {
	return secretKeyPattern.MatchString(key)
}

// maskedValue is the shared masking rule for any single key/value pair -
// used by kvTable rows (params/headers-shaped tables) directly, and by
// maskSecrets below for free-text bodies.
func maskedValue(key, value string, reveal bool) string {
	if reveal || value == "" || !looksLikeSecretKey(key) {
		return value
	}
	return "****"
}

// jsonSecretPairPattern matches a pretty-printed JSON `"key": "value"` pair,
// capturing the key and value separately. Applied before HighlightJSON -
// ANSI styling wraps the key and value in separate escape sequences, which
// would otherwise break this match.
var jsonSecretPairPattern = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"\s*:\s*"((?:[^"\\]|\\.)*)"`)

// plainSecretLinePattern matches a plain `key: value` or `key=value` line
// (headers, form-urlencoded text, log-style output), capturing the key and
// the rest of the line as the value.
var plainSecretLinePattern = regexp.MustCompile(`(?m)^(\s*)([\w.-]+)(\s*[:=]\s*)(.+)$`)

// maskSecrets masks secret-looking key/value pairs in rendered request or
// response body text, unless reveal is true. It recognizes two shapes: a
// pretty-printed JSON "key": "value" pair, and a plain key: value / key=value
// line - the latter never matches a JSON line (those start with a quote, not
// a bare identifier), so both passes can run unconditionally without
// double-masking or clobbering each other.
func maskSecrets(text string, reveal bool) string {
	if reveal {
		return text
	}
	text = jsonSecretPairPattern.ReplaceAllStringFunc(text, func(match string) string {
		sub := jsonSecretPairPattern.FindStringSubmatch(match)
		key, value := sub[1], sub[2]
		if !looksLikeSecretKey(key) {
			return match
		}
		return `"` + key + `": "` + maskedValue(key, value, false) + `"`
	})
	text = plainSecretLinePattern.ReplaceAllStringFunc(text, func(match string) string {
		sub := plainSecretLinePattern.FindStringSubmatch(match)
		indent, key, sep, value := sub[1], sub[2], sub[3], sub[4]
		if !looksLikeSecretKey(key) {
			return match
		}
		return indent + key + sep + maskedValue(key, strings.TrimRight(value, "\r"), false)
	})
	return text
}
