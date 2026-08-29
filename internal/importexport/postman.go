package importexport

import (
	"encoding/json"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"net/url"
	"strings"
)

const postmanSchemaV21 = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

type postmanCollection struct {
	Info postmanInfo   `json:"info"`
	Item []postmanItem `json:"item"`
}

type postmanInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item,omitempty"`
	Request *postmanRequest `json:"request,omitempty"`
	Event   []postmanEvent  `json:"event,omitempty"`
}

type postmanRequest struct {
	Method string          `json:"method"`
	Header []postmanHeader `json:"header,omitempty"`
	URL    postmanURL      `json:"url"`
	Body   *postmanBody    `json:"body,omitempty"`
}

type postmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type postmanBody struct {
	Mode string `json:"mode,omitempty"`
	Raw  string `json:"raw,omitempty"`
}

type postmanEvent struct {
	Listen string        `json:"listen"`
	Script postmanScript `json:"script"`
}

type postmanScript struct {
	Exec []string `json:"exec,omitempty"`
}

type postmanURL struct {
	Raw   string              `json:"raw"`
	Query []postmanQueryParam `json:"query,omitempty"`
}

// UnmarshalJSON accepts both Postman URL shapes: a plain string, or the
// richer {raw, query, ...} object.
func (u *postmanURL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		u.Raw = s
		return nil
	}
	type alias postmanURL
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*u = postmanURL(a)
	return nil
}

type postmanQueryParam struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

// ParsePostmanCollection parses a Postman Collection v2.1 JSON export into
// an ImportedNode tree.
func ParsePostmanCollection(data []byte) (ImportedNode, error) {
	var col postmanCollection
	if err := json.Unmarshal(data, &col); err != nil {
		return ImportedNode{}, err
	}
	root := ImportedNode{Name: col.Info.Name}
	for _, item := range col.Item {
		root.Children = append(root.Children, convertPostmanItem(item))
	}
	return root, nil
}

func convertPostmanItem(item postmanItem) ImportedNode {
	if item.Request == nil {
		node := ImportedNode{Name: item.Name}
		for _, child := range item.Item {
			node.Children = append(node.Children, convertPostmanItem(child))
		}
		return node
	}

	req := collection.Request{
		Method:          collection.Method(strings.ToUpper(item.Request.Method)),
		FollowRedirects: true,
	}
	for _, h := range item.Request.Header {
		req.Headers = append(req.Headers, collection.Header{Key: h.Key, Value: h.Value, Enabled: !h.Disabled})
	}

	rawURL := item.Request.URL.Raw
	if u, err := url.Parse(rawURL); err == nil && u.RawQuery != "" {
		req.Params = collection.ParseQueryParams(rawURL)
		u.RawQuery = ""
		req.URL = u.String()
	} else {
		req.URL = rawURL
	}
	for _, q := range item.Request.URL.Query {
		if !hasParam(req.Params, q.Key) {
			req.Params = append(req.Params, collection.QueryParam{Key: q.Key, Value: q.Value, Enabled: !q.Disabled})
		}
	}

	if item.Request.Body != nil && item.Request.Body.Mode == "raw" {
		req.Body = collection.Body{Type: collection.BodyRaw, RawText: item.Request.Body.Raw, RawContentType: findHeader(req.Headers, "Content-Type")}
	} else {
		req.Body = collection.Body{Type: collection.BodyNone}
	}

	for _, ev := range item.Event {
		script := strings.Join(ev.Script.Exec, "\n")
		switch ev.Listen {
		case "prerequest":
			req.PreRequestScript = script
		case "test":
			req.TestScript = script
		}
	}

	return ImportedNode{Name: item.Name, Request: &req}
}

func hasParam(params []collection.QueryParam, key string) bool {
	for _, p := range params {
		if p.Key == key {
			return true
		}
	}
	return false
}

// ExportPostmanCollection builds a Postman Collection v2.1 JSON document
// from an ImportedNode tree (e.g. produced by persist.go's BuildImportedNode).
func ExportPostmanCollection(name string, root ImportedNode) ([]byte, error) {
	col := postmanCollection{Info: postmanInfo{Name: name, Schema: postmanSchemaV21}}
	for _, child := range root.Children {
		col.Item = append(col.Item, convertToPostmanItem(child))
	}
	return json.MarshalIndent(col, "", "  ")
}

func convertToPostmanItem(node ImportedNode) postmanItem {
	if node.Request == nil {
		item := postmanItem{Name: node.Name}
		for _, child := range node.Children {
			item.Item = append(item.Item, convertToPostmanItem(child))
		}
		return item
	}

	req := node.Request
	urlStr, err := collection.BuildURL(req.URL, req.Params)
	if err != nil {
		urlStr = req.URL
	}
	pmReq := &postmanRequest{Method: string(req.Method), URL: postmanURL{Raw: urlStr}}
	for _, h := range req.Headers {
		pmReq.Header = append(pmReq.Header, postmanHeader{Key: h.Key, Value: h.Value, Disabled: !h.Enabled})
	}
	for _, p := range req.Params {
		pmReq.URL.Query = append(pmReq.URL.Query, postmanQueryParam{Key: p.Key, Value: p.Value, Disabled: !p.Enabled})
	}
	if req.Body.Type == collection.BodyRaw {
		pmReq.Body = &postmanBody{Mode: "raw", Raw: req.Body.RawText}
	}

	item := postmanItem{Name: node.Name, Request: pmReq}
	if req.PreRequestScript != "" {
		item.Event = append(item.Event, postmanEvent{Listen: "prerequest", Script: postmanScript{Exec: strings.Split(req.PreRequestScript, "\n")}})
	}
	if req.TestScript != "" {
		item.Event = append(item.Event, postmanEvent{Listen: "test", Script: postmanScript{Exec: strings.Split(req.TestScript, "\n")}})
	}
	return item
}
