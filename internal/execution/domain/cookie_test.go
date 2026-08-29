package execution

import "testing"

func TestParseSetCookie_NameAndValue(t *testing.T) {
	c := ParseSetCookie("session=abc123")
	if c.Name != "session" || c.Value != "abc123" {
		t.Errorf("got %+v", c)
	}
}

func TestParseSetCookie_AttributesParsed(t *testing.T) {
	c := ParseSetCookie("session=abc123; Domain=example.com; Path=/; Expires=Wed, 09 Jun 2021 10:18:14 GMT; Secure; HttpOnly")
	if c.Name != "session" || c.Value != "abc123" {
		t.Fatalf("name/value = %+v", c)
	}
	if c.Domain != "example.com" {
		t.Errorf("domain = %q", c.Domain)
	}
	if c.Path != "/" {
		t.Errorf("path = %q", c.Path)
	}
	if c.Expires != "Wed, 09 Jun 2021 10:18:14 GMT" {
		t.Errorf("expires = %q", c.Expires)
	}
	if !c.Secure {
		t.Error("expected Secure = true")
	}
	if !c.HTTPOnly {
		t.Error("expected HTTPOnly = true")
	}
}

func TestParseSetCookie_FlagsDefaultFalseWhenAbsent(t *testing.T) {
	c := ParseSetCookie("session=abc123")
	if c.Secure || c.HTTPOnly {
		t.Errorf("expected flags false by default, got %+v", c)
	}
}

func TestParseSetCookie_EmptyHeaderYieldsEmptyCookie(t *testing.T) {
	c := ParseSetCookie("")
	if c.Name != "" || c.Value != "" {
		t.Errorf("got %+v, want empty", c)
	}
}
