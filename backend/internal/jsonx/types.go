// Package jsonx holds scalar types that serialise the way the former Laravel
// API did, so the React storefront keeps working unchanged: decimal columns
// become fixed-scale strings and timestamps use Laravel's ISO-8601 layout.
package jsonx

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Money is a numeric(18,4) column: JSON "12.5000" (Eloquent's decimal:4 cast).
type Money float64

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatFloat(float64(m), 'f', 4, 64))
}

func (m *Money) UnmarshalJSON(b []byte) error { return unmarshalDecimal(b, (*float64)(m)) }

func (m *Money) ScanNumeric(v pgtype.Numeric) error {
	f, err := numericToFloat(v)
	*m = Money(f)
	return err
}

func (m Money) Float() float64 { return float64(m) }

// Rate is a numeric(18,8) column: JSON "0.92000000" (Eloquent's decimal:8 cast).
type Rate float64

func (r Rate) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatFloat(float64(r), 'f', 8, 64))
}

func (r *Rate) UnmarshalJSON(b []byte) error { return unmarshalDecimal(b, (*float64)(r)) }

func (r *Rate) ScanNumeric(v pgtype.Numeric) error {
	f, err := numericToFloat(v)
	*r = Rate(f)
	return err
}

func (r Rate) Float() float64 { return float64(r) }

func numericToFloat(v pgtype.Numeric) (float64, error) {
	if !v.Valid {
		return 0, nil
	}
	f, err := v.Float64Value()
	if err != nil {
		return 0, err
	}
	return f.Float64, nil
}

func unmarshalDecimal(b []byte, dst *float64) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*dst = f
		return nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	*dst = f
	return nil
}

// Time is a timestamp column rendered as "2026-06-19T10:00:00.000000Z".
type Time struct {
	time.Time
}

const layout = "2006-01-02T15:04:05.000000Z07:00"

func Now() Time { return Time{time.Now().UTC()} }

func From(t time.Time) Time { return Time{t.UTC()} }

func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.UTC().Format(layout))
}

func (t *Time) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	t.Time = parsed.UTC()
	return nil
}

func (t *Time) ScanTimestamp(v pgtype.Timestamp) error {
	if !v.Valid {
		t.Time = time.Time{}
		return nil
	}
	t.Time = v.Time.UTC()
	return nil
}

func (t *Time) ScanTimestamptz(v pgtype.Timestamptz) error {
	if !v.Valid {
		t.Time = time.Time{}
		return nil
	}
	t.Time = v.Time.UTC()
	return nil
}

func (t Time) TimestampValue() (pgtype.Timestamp, error) {
	return pgtype.Timestamp{Time: t.Time.UTC(), Valid: !t.IsZero()}, nil
}

// JSONMap is a json/jsonb column holding an object (or NULL).
type JSONMap map[string]any
