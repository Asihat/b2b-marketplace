// Package validate is a small request validator producing Laravel-shaped
// error payloads: {"message": "The name field is required.", "errors": {...}}.
package validate

import (
	"fmt"
	"math"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
)

// V validates one input map. Methods record errors and return the parsed
// value plus ok=true when a usable (non-null, valid) value is present.
type V struct {
	data   map[string]any
	errs   map[string][]string
	order  []string
	root   *V
	prefix string
}

func New(data map[string]any) *V {
	if data == nil {
		data = map[string]any{}
	}
	return &V{data: data, errs: map[string][]string{}}
}

func (v *V) Data() map[string]any { return v.data }

// Has reports whether the key was sent at all (null counts as sent).
func (v *V) Has(key string) bool {
	_, ok := v.data[key]
	return ok
}

// Filled reports whether the key was sent with a non-null value.
func (v *V) Filled(key string) bool {
	val, ok := v.data[key]
	return ok && val != nil
}

func (v *V) Get(key string) any { return v.data[key] }

// Fail records a custom error for a key.
func (v *V) Fail(key, msg string) {
	if v.root != nil {
		v.root.Fail(v.prefix+"."+key, msg)
		return
	}
	if _, seen := v.errs[key]; !seen {
		v.order = append(v.order, key)
	}
	v.errs[key] = append(v.errs[key], msg)
}

func (v *V) HasErrors() bool {
	if v.root != nil {
		return v.root.HasErrors()
	}
	return len(v.errs) > 0
}

// Err returns nil or a 422 error with the collected messages.
func (v *V) Err() error {
	if v.root != nil {
		return v.root.Err()
	}
	if len(v.errs) == 0 {
		return nil
	}
	first := v.errs[v.order[0]][0]
	total := 0
	for _, msgs := range v.errs {
		total += len(msgs)
	}
	msg := first
	if total > 1 {
		if total == 2 {
			msg += " (and 1 more error)"
		} else {
			msg += fmt.Sprintf(" (and %d more errors)", total-1)
		}
	}
	return &httpx.Error{Status: http.StatusUnprocessableEntity, Message: msg, Errors: v.errs}
}

func attr(key string) string {
	return strings.ReplaceAll(key, "_", " ")
}

// presence handles the shared required / nullable / missing logic. The
// returned bool tells callers to continue type checking.
func (v *V) presence(key string, required, nullable bool, typeMsg string) (any, bool) {
	val, ok := v.data[key]
	if !ok {
		if required {
			v.Fail(key, fmt.Sprintf("The %s field is required.", attr(key)))
		}
		return nil, false
	}
	if val == nil {
		switch {
		case required:
			v.Fail(key, fmt.Sprintf("The %s field is required.", attr(key)))
		case nullable:
		default:
			v.Fail(key, typeMsg)
		}
		return nil, false
	}
	return val, true
}

// Str describes rules for string fields.
type Str struct {
	Required bool
	Nullable bool
	Email    bool
	Max      int
	Min      int
	Size     int
	In       []string
}

// String validates a string field.
func (v *V) String(key string, o Str) (string, bool) {
	val, ok := v.presence(key, o.Required, o.Nullable, fmt.Sprintf("The %s field must be a string.", attr(key)))
	if !ok {
		return "", false
	}
	s, isStr := val.(string)
	if !isStr {
		if f, isNum := val.(float64); isNum {
			s = strconv.FormatFloat(f, 'f', -1, 64)
		} else {
			v.Fail(key, fmt.Sprintf("The %s field must be a string.", attr(key)))
			return "", false
		}
	}
	n := utf8.RuneCountInString(s)
	switch {
	case o.Size > 0 && n != o.Size:
		v.Fail(key, fmt.Sprintf("The %s field must be %d characters.", attr(key), o.Size))
		return "", false
	case o.Max > 0 && n > o.Max:
		v.Fail(key, fmt.Sprintf("The %s field must not be greater than %d characters.", attr(key), o.Max))
		return "", false
	case o.Min > 0 && n < o.Min:
		v.Fail(key, fmt.Sprintf("The %s field must be at least %d characters.", attr(key), o.Min))
		return "", false
	}
	if o.Email {
		if a, err := mail.ParseAddress(s); err != nil || a.Address != s || !strings.Contains(s, "@") {
			v.Fail(key, fmt.Sprintf("The %s field must be a valid email address.", attr(key)))
			return "", false
		}
	}
	if len(o.In) > 0 && !contains(o.In, s) {
		v.Fail(key, fmt.Sprintf("The selected %s is invalid.", attr(key)))
		return "", false
	}
	return s, true
}

// OptString validates a nullable string and reports whether the key was sent
// (with either a valid value or null) so partial updates can null it out.
func (v *V) OptString(key string, o Str) (*string, bool) {
	o.Nullable = true
	if !v.Has(key) {
		if o.Required {
			v.Fail(key, fmt.Sprintf("The %s field is required.", attr(key)))
		}
		return nil, false
	}
	if v.data[key] == nil {
		if o.Required {
			v.Fail(key, fmt.Sprintf("The %s field is required.", attr(key)))
			return nil, false
		}
		return nil, true
	}
	s, ok := v.String(key, o)
	if !ok {
		return nil, false
	}
	return &s, true
}

// Num describes rules for integer and float fields.
type Num struct {
	Required bool
	Nullable bool
	Min      *float64
	Max      *float64
}

func F(f float64) *float64 { return &f }

func toFloat(val any) (float64, bool) {
	switch t := val.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	case bool:
		return 0, false
	}
	return 0, false
}

// Float validates a numeric field (numbers or numeric strings).
func (v *V) Float(key string, o Num) (float64, bool) {
	val, ok := v.presence(key, o.Required, o.Nullable, fmt.Sprintf("The %s field must be a number.", attr(key)))
	if !ok {
		return 0, false
	}
	f, isNum := toFloat(val)
	if !isNum {
		v.Fail(key, fmt.Sprintf("The %s field must be a number.", attr(key)))
		return 0, false
	}
	if o.Min != nil && f < *o.Min {
		v.Fail(key, fmt.Sprintf("The %s field must be at least %s.", attr(key), fmtNum(*o.Min)))
		return 0, false
	}
	if o.Max != nil && f > *o.Max {
		v.Fail(key, fmt.Sprintf("The %s field must not be greater than %s.", attr(key), fmtNum(*o.Max)))
		return 0, false
	}
	return f, true
}

// Int validates an integer field.
func (v *V) Int(key string, o Num) (int64, bool) {
	val, ok := v.presence(key, o.Required, o.Nullable, fmt.Sprintf("The %s field must be an integer.", attr(key)))
	if !ok {
		return 0, false
	}
	f, isNum := toFloat(val)
	if !isNum || f != math.Trunc(f) || math.Abs(f) > math.MaxInt64 {
		v.Fail(key, fmt.Sprintf("The %s field must be an integer.", attr(key)))
		return 0, false
	}
	if o.Min != nil && f < *o.Min {
		v.Fail(key, fmt.Sprintf("The %s field must be at least %s.", attr(key), fmtNum(*o.Min)))
		return 0, false
	}
	if o.Max != nil && f > *o.Max {
		v.Fail(key, fmt.Sprintf("The %s field must not be greater than %s.", attr(key), fmtNum(*o.Max)))
		return 0, false
	}
	return int64(f), true
}

// OptInt validates a nullable integer; the bool reports whether the key was sent.
func (v *V) OptInt(key string, o Num) (*int64, bool) {
	o.Nullable = true
	if !v.Has(key) {
		return nil, false
	}
	if v.data[key] == nil {
		return nil, true
	}
	n, ok := v.Int(key, o)
	if !ok {
		return nil, false
	}
	return &n, true
}

// Bool validates Laravel's "boolean" rule: true/false/1/0/"1"/"0".
func (v *V) Bool(key string, required bool) (bool, bool) {
	val, ok := v.presence(key, required, false, fmt.Sprintf("The %s field must be true or false.", attr(key)))
	if !ok {
		return false, false
	}
	switch t := val.(type) {
	case bool:
		return t, true
	case float64:
		if t == 1 {
			return true, true
		}
		if t == 0 {
			return false, true
		}
	case string:
		switch strings.ToLower(t) {
		case "1", "true":
			return true, true
		case "0", "false":
			return false, true
		}
	}
	v.Fail(key, fmt.Sprintf("The %s field must be true or false.", attr(key)))
	return false, false
}

// Arr describes rules for array fields.
type Arr struct {
	Required bool
	Present  bool // 'present': key must exist, but may be empty
	Nullable bool
	Min      int
	Max      int
}

// Array validates a list field.
func (v *V) Array(key string, o Arr) ([]any, bool) {
	if o.Present && !v.Has(key) {
		v.Fail(key, fmt.Sprintf("The %s field must be present.", attr(key)))
		return nil, false
	}
	val, ok := v.presence(key, o.Required, o.Nullable, fmt.Sprintf("The %s field must be an array.", attr(key)))
	if !ok {
		return nil, false
	}
	list, isList := val.([]any)
	if !isList {
		v.Fail(key, fmt.Sprintf("The %s field must be an array.", attr(key)))
		return nil, false
	}
	if o.Required && len(list) == 0 {
		v.Fail(key, fmt.Sprintf("The %s field is required.", attr(key)))
		return nil, false
	}
	if o.Min > 0 && len(list) < o.Min {
		v.Fail(key, fmt.Sprintf("The %s field must have at least %d items.", attr(key), o.Min))
		return nil, false
	}
	if o.Max > 0 && len(list) > o.Max {
		v.Fail(key, fmt.Sprintf("The %s field must not have more than %d items.", attr(key), o.Max))
		return nil, false
	}
	return list, true
}

// Object validates a nested object field (e.g. name_translations).
func (v *V) Object(key string, nullable bool) (map[string]any, bool) {
	val, ok := v.presence(key, false, nullable, fmt.Sprintf("The %s field must be an array.", attr(key)))
	if !ok {
		return nil, false
	}
	obj, isObj := val.(map[string]any)
	if !isObj {
		v.Fail(key, fmt.Sprintf("The %s field must be an array.", attr(key)))
		return nil, false
	}
	return obj, true
}

// Sub returns a validator over a nested object so list rows can be checked
// with the same rules; errors are reported under "prefix.key".
func (v *V) Sub(prefix string, data map[string]any) *V {
	root := v
	if v.root != nil {
		root = v.root
		prefix = v.prefix + "." + prefix
	}
	if data == nil {
		data = map[string]any{}
	}
	return &V{data: data, root: root, prefix: prefix}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func fmtNum(f float64) string {
	if f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// SortedKeys is a helper for deterministic error ordering in tests.
func SortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
