package execution

import "strings"

// Cookie is a parsed Set-Cookie response header.
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  string
	Secure   bool
	HTTPOnly bool
}

// ParseSetCookie parses a single Set-Cookie header value into a Cookie.
// Unknown attributes are ignored.
func ParseSetCookie(header string) Cookie {
	var c Cookie
	for i, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, _ := strings.Cut(part, "=")
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		if i == 0 {
			c.Name, c.Value = key, val
			continue
		}
		switch strings.ToLower(key) {
		case "domain":
			c.Domain = val
		case "path":
			c.Path = val
		case "expires":
			c.Expires = val
		case "secure":
			c.Secure = true
		case "httponly":
			c.HTTPOnly = true
		}
	}
	return c
}
