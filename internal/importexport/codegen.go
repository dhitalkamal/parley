package importexport

import (
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strconv"
	"strings"
)

// GenerateCurl emits a curl command equivalent to req - only enabled
// headers/params, single-quoted and escaped for a POSIX shell.
func GenerateCurl(req collection.Request) string {
	urlStr, err := collection.BuildURL(req.URL, req.Params)
	if err != nil {
		urlStr = req.URL
	}

	var b strings.Builder
	fmt.Fprintf(&b, "curl -X %s %s", req.Method, shellQuote(urlStr))
	headers := collection.EffectiveHeaders(req.Headers)
	hasContentType := false
	for _, h := range headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			hasContentType = true
		}
		fmt.Fprintf(&b, " \\\n  -H %s", shellQuote(h.Key+": "+h.Value))
	}
	// httpclient.Client.Do auto-sets Content-Type from the body's raw
	// content type whenever it isn't already set by an explicit header - so
	// the generated command must too, or it won't match what actually goes
	// out on the wire.
	if req.Body.Type == collection.BodyRaw && req.Body.RawContentType != "" && !hasContentType {
		fmt.Fprintf(&b, " \\\n  -H %s", shellQuote("Content-Type: "+req.Body.RawContentType))
	}
	if req.Body.Type == collection.BodyRaw && req.Body.RawText != "" {
		fmt.Fprintf(&b, " \\\n  -d %s", shellQuote(req.Body.RawText))
	}
	if req.InsecureSkipVerify {
		b.WriteString(" \\\n  -k")
	}
	return b.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// GenerateGo emits a standalone net/http Go program equivalent to req.
func GenerateGo(req collection.Request) string {
	urlStr, err := collection.BuildURL(req.URL, req.Params)
	if err != nil {
		urlStr = req.URL
	}
	hasBody := req.Body.Type == collection.BodyRaw && req.Body.RawText != ""

	var b strings.Builder
	b.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n")
	if hasBody {
		b.WriteString("\t\"strings\"\n")
	}
	b.WriteString(")\n\nfunc main() {\n")

	if hasBody {
		fmt.Fprintf(&b, "\tbody := strings.NewReader(%s)\n", strconv.Quote(req.Body.RawText))
		fmt.Fprintf(&b, "\treq, err := http.NewRequest(%s, %s, body)\n", strconv.Quote(string(req.Method)), strconv.Quote(urlStr))
	} else {
		fmt.Fprintf(&b, "\treq, err := http.NewRequest(%s, %s, nil)\n", strconv.Quote(string(req.Method)), strconv.Quote(urlStr))
	}
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")

	headers := collection.EffectiveHeaders(req.Headers)
	hasContentType := false
	for _, h := range headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			hasContentType = true
		}
		fmt.Fprintf(&b, "\treq.Header.Set(%s, %s)\n", strconv.Quote(h.Key), strconv.Quote(h.Value))
	}
	if hasBody && req.Body.RawContentType != "" && !hasContentType {
		fmt.Fprintf(&b, "\treq.Header.Set(\"Content-Type\", %s)\n", strconv.Quote(req.Body.RawContentType))
	}

	b.WriteString("\n\tresp, err := http.DefaultClient.Do(req)\n")
	b.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	b.WriteString("\tdefer resp.Body.Close()\n\n")
	b.WriteString("\trespBody, _ := io.ReadAll(resp.Body)\n")
	b.WriteString("\tfmt.Println(resp.Status)\n")
	b.WriteString("\tfmt.Println(string(respBody))\n")
	b.WriteString("}\n")

	return b.String()
}
