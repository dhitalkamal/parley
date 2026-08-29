package environment

import "sort"

// ApplyVariableUpdates merges script-produced key->value updates into an
// existing Variable list: an update to an already-present key changes its
// Value in place, preserving Enabled/Secret; a key that didn't exist before
// is appended as a new enabled, non-secret Variable. New keys are appended
// in sorted order for deterministic output. existing is left untouched.
func ApplyVariableUpdates(existing []Variable, updates map[string]string) []Variable {
	out := make([]Variable, len(existing))
	copy(out, existing)

	seen := make(map[string]bool, len(updates))
	for i, v := range out {
		if newVal, ok := updates[v.Key]; ok {
			out[i].Value = newVal
			seen[v.Key] = true
		}
	}

	var newKeys []string
	for k := range updates {
		if !seen[k] {
			newKeys = append(newKeys, k)
		}
	}
	sort.Strings(newKeys)
	for _, k := range newKeys {
		out = append(out, Variable{Key: k, Value: updates[k], Enabled: true})
	}
	return out
}
