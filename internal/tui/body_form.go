package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
)

// fileFieldPrefix marks a multipart form field's value as a file path
// rather than literal text - the same "@" convention curl's -F flag uses
// (-F "file=@/path/to/file.png"). It lets URLEncoded and Multipart share
// kvTable as-is instead of a new field-type-aware widget: kvTable only
// ever deals in plain key/value/enabled rows, and this prefix is applied
// and stripped at the boundary between it and collection.BodyFormField.
const fileFieldPrefix = "@"

// formFieldsFromRows converts kvTable's rows into collection.BodyFormField. The
// "@" file convention only applies when allowFiles is true (Multipart) -
// URLEncoded has no notion of a file field, and the httpclient never
// treats "@" specially for that type either, so a literal "@" in a
// URLEncoded value round-trips as plain text instead.
func formFieldsFromRows(rows []kvRow, allowFiles bool) []collection.BodyFormField {
	out := make([]collection.BodyFormField, len(rows))
	for i, r := range rows {
		if allowFiles {
			if path, ok := strings.CutPrefix(r.Value, fileFieldPrefix); ok {
				out[i] = collection.BodyFormField{Key: r.Key, FilePath: path, IsFile: true, Enabled: r.Enabled}
				continue
			}
		}
		out[i] = collection.BodyFormField{Key: r.Key, Value: r.Value, Enabled: r.Enabled}
	}
	return out
}

// rowsFromFormFields is formFieldsFromRows's inverse, used by SetBody to
// load a saved request back into the shared kvTable editor.
func rowsFromFormFields(fields []collection.BodyFormField) []kvRow {
	out := make([]kvRow, len(fields))
	for i, f := range fields {
		value := f.Value
		if f.IsFile {
			value = fileFieldPrefix + f.FilePath
		}
		out[i] = kvRow{Key: f.Key, Value: value, Enabled: f.Enabled}
	}
	return out
}
