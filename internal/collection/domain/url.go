package collection

import (
	"net/url"
	"strings"
)

// BuildURL merges the enabled query params into rawURL, replacing any existing
// query string. Disabled params are dropped so the table stays the source of
// truth. Params are encoded in row order rather than url.Values' alphabetical
// order, so the editor's order matches what gets sent.
//
// The base (everything before "?"/"#") is kept as the literal text the user
// typed rather than round-tripped through url.URL.String(), which would
// percent-encode characters like "{" and "}" - and so would mangle an
// unresolved {{var}} placeholder into %7B%7Bvar%7D%7D on the wire instead of
// leaving it as the obviously-broken literal it actually is.
func BuildURL(rawURL string, params []QueryParam) (string, error) {
	if _, err := url.Parse(rawURL); err != nil {
		return "", err
	}
	base := rawURL
	if i := strings.IndexAny(base, "?#"); i >= 0 {
		base = base[:i]
	}

	var q strings.Builder
	for _, p := range params {
		if !p.Enabled {
			continue
		}
		if q.Len() > 0 {
			q.WriteByte('&')
		}
		q.WriteString(url.QueryEscape(p.Key))
		q.WriteByte('=')
		q.WriteString(url.QueryEscape(p.Value))
	}
	if q.Len() == 0 {
		return base, nil
	}
	return base + "?" + q.String(), nil
}

// ParseQueryParams reads the existing query string off rawURL so the params
// table can be initialized when a user pastes a URL directly.
func ParseQueryParams(rawURL string) []QueryParam {
	u, err := url.Parse(rawURL)
	if err != nil || u.RawQuery == "" {
		return nil
	}
	var params []QueryParam
	for _, pair := range strings.Split(u.RawQuery, "&") {
		key, val, _ := strings.Cut(pair, "=")
		k, err1 := url.QueryUnescape(key)
		v, err2 := url.QueryUnescape(val)
		if err1 != nil || err2 != nil {
			continue
		}
		params = append(params, QueryParam{Key: k, Value: v, Enabled: true})
	}
	return params
}
