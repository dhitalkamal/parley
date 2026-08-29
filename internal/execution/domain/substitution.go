package execution

import (
	"regexp"
	"sort"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
)

var variablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// Substitute replaces {{key}} placeholders in text using vars. A placeholder
// whose key isn't found is left in the output untouched and its key
// reported in unresolved, so the caller can warn without blocking the send.
func Substitute(text string, vars map[string]string) (result string, unresolved []string) {
	result = variablePattern.ReplaceAllStringFunc(text, func(match string) string {
		key := variablePattern.FindStringSubmatch(match)[1]
		if v, ok := vars[key]; ok {
			return v
		}
		unresolved = append(unresolved, key)
		return match
	})
	return result, unresolved
}

// ResolveVariables merges globals and the active environment into a single
// lookup keyed by variable name, with the environment overriding globals
// for the same key. Disabled variables are excluded from the result.
func ResolveVariables(globals, env environment.Environment) map[string]environment.Variable {
	out := make(map[string]environment.Variable)
	for _, v := range globals.Variables {
		if v.Enabled {
			out[v.Key] = v
		}
	}
	for _, v := range env.Variables {
		if v.Enabled {
			out[v.Key] = v
		}
	}
	return out
}

// FlattenVariables extracts a plain key->value lookup from resolved
// variables, for feeding into Substitute.
func FlattenVariables(vars map[string]environment.Variable) map[string]string {
	out := make(map[string]string, len(vars))
	for k, v := range vars {
		out[k] = v.Value
	}
	return out
}

// SubstituteRequest resolves {{var}} placeholders in a collection.Request's URL,
// params (key and value), headers (key and value), and raw body text. It
// returns a resolved copy - the original collection.Request and its slices are left
// untouched - plus the sorted, deduplicated list of variable keys that
// couldn't be resolved.
func SubstituteRequest(req collection.Request, vars map[string]environment.Variable) (collection.Request, []string) {
	flat := FlattenVariables(vars)
	seen := make(map[string]bool)
	var unresolved []string
	record := func(keys []string) {
		for _, k := range keys {
			if !seen[k] {
				seen[k] = true
				unresolved = append(unresolved, k)
			}
		}
	}

	out := req
	var u []string
	out.URL, u = Substitute(req.URL, flat)
	record(u)

	out.Params = make([]collection.QueryParam, len(req.Params))
	for i, p := range req.Params {
		var uk, uv []string
		out.Params[i] = p
		out.Params[i].Key, uk = Substitute(p.Key, flat)
		out.Params[i].Value, uv = Substitute(p.Value, flat)
		record(uk)
		record(uv)
	}

	out.Headers = make([]collection.Header, len(req.Headers))
	for i, h := range req.Headers {
		var uk, uv []string
		out.Headers[i] = h
		out.Headers[i].Key, uk = Substitute(h.Key, flat)
		out.Headers[i].Value, uv = Substitute(h.Value, flat)
		record(uk)
		record(uv)
	}

	out.Body.RawText, u = Substitute(req.Body.RawText, flat)
	record(u)

	out.Body.FormFields = make([]collection.BodyFormField, len(req.Body.FormFields))
	for i, ff := range req.Body.FormFields {
		var uk, uv []string
		out.Body.FormFields[i] = ff
		out.Body.FormFields[i].Key, uk = Substitute(ff.Key, flat)
		out.Body.FormFields[i].Value, uv = Substitute(ff.Value, flat)
		record(uk)
		record(uv)
	}

	out.Body.GraphQLQuery, u = Substitute(req.Body.GraphQLQuery, flat)
	record(u)
	out.Body.GraphQLVariables, u = Substitute(req.Body.GraphQLVariables, flat)
	record(u)

	sort.Strings(unresolved)
	return out, unresolved
}
