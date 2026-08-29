package importexport

import (
	"encoding/base64"
	"errors"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"net/url"
	"strings"
)

// ParseCurl parses a curl command line into a collection.Request. It covers the
// flags that matter for API testing (-X/--request, -H/--header, -d and its
// variants, -u/--user, -k/--insecure, --url) and the common "copy as cURL"
// shapes (quoted args, backslash line continuations). It does not attempt
// full curl compatibility - e.g. -G/--get (query-string-from-data), config
// files, and upload flags aren't supported.
func ParseCurl(command string) (collection.Request, error) {
	command = strings.ReplaceAll(command, "\\\r\n", " ")
	command = strings.ReplaceAll(command, "\\\n", " ")

	tokens, err := shellSplit(strings.TrimSpace(command))
	if err != nil {
		return collection.Request{}, err
	}
	if len(tokens) == 0 {
		return collection.Request{}, errors.New("empty command")
	}
	if strings.ToLower(tokens[0]) != "curl" {
		return collection.Request{}, errors.New("not a curl command (must start with \"curl\")")
	}
	tokens = tokens[1:]

	var (
		method    string
		rawURL    string
		headers   []collection.Header
		dataParts []string
		userPass  string
		insecure  bool
	)

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		next := func() (string, bool) {
			if i+1 < len(tokens) {
				i++
				return tokens[i], true
			}
			return "", false
		}
		switch tok {
		case "-X", "--request":
			if v, ok := next(); ok {
				method = v
			}
		case "-H", "--header":
			if v, ok := next(); ok {
				if key, val, found := strings.Cut(v, ":"); found {
					headers = append(headers, collection.Header{Key: strings.TrimSpace(key), Value: strings.TrimSpace(val), Enabled: true})
				}
			}
		case "-d", "--data", "--data-raw", "--data-binary", "--data-ascii":
			if v, ok := next(); ok {
				dataParts = append(dataParts, v)
			}
		case "-u", "--user":
			if v, ok := next(); ok {
				userPass = v
			}
		case "--url":
			if v, ok := next(); ok {
				rawURL = v
			}
		case "-A", "--user-agent":
			if v, ok := next(); ok {
				headers = append(headers, collection.Header{Key: "User-Agent", Value: v, Enabled: true})
			}
		case "-k", "--insecure":
			insecure = true
		default:
			if !strings.HasPrefix(tok, "-") && rawURL == "" {
				rawURL = tok
			}
		}
	}

	if rawURL == "" {
		return collection.Request{}, errors.New("curl command has no URL")
	}

	bodyText := strings.Join(dataParts, "&")
	contentType := findHeader(headers, "Content-Type")
	if contentType == "" && bodyText != "" {
		trimmed := strings.TrimSpace(bodyText)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			contentType = "application/json"
		} else {
			contentType = "application/x-www-form-urlencoded"
		}
	}

	if method == "" {
		if bodyText != "" {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	req := collection.Request{
		Method:             collection.Method(strings.ToUpper(method)),
		Headers:            headers,
		FollowRedirects:    true,
		InsecureSkipVerify: insecure,
	}
	if bodyText != "" {
		req.Body = collection.Body{Type: collection.BodyRaw, RawContentType: contentType, RawText: bodyText}
	} else {
		req.Body = collection.Body{Type: collection.BodyNone}
	}
	if userPass != "" {
		user, pass, _ := strings.Cut(userPass, ":")
		encoded := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		req.Headers = append(req.Headers, collection.Header{Key: "Authorization", Value: "Basic " + encoded, Enabled: true})
	}

	if parsed, err := url.Parse(rawURL); err == nil && parsed.RawQuery != "" {
		req.Params = collection.ParseQueryParams(rawURL)
		parsed.RawQuery = ""
		req.URL = parsed.String()
	} else {
		req.URL = rawURL
	}

	return req, nil
}

func findHeader(headers []collection.Header, key string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Key, key) {
			return h.Value
		}
	}
	return ""
}
