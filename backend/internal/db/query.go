package db

import (
	"fmt"
	"strings"
)

// Query accumulates WHERE conditions and positional arguments for dynamic
// listing endpoints (search, filters, sorting, pagination).
type Query struct {
	where []string
	args  []any
}

// Arg registers a bind value and returns its $n placeholder.
func (q *Query) Arg(v any) string {
	q.args = append(q.args, v)
	return fmt.Sprintf("$%d", len(q.args))
}

// Where adds a condition. Use Arg() inside the condition for parameters.
func (q *Query) Where(cond string) *Query {
	q.where = append(q.where, cond)
	return q
}

// Wheref adds a condition built from a format string whose %s verbs are
// replaced by the placeholders of the given arguments.
func (q *Query) Wheref(format string, args ...any) *Query {
	ph := make([]any, len(args))
	for i, a := range args {
		ph[i] = q.Arg(a)
	}
	return q.Where(fmt.Sprintf(format, ph...))
}

// ILike adds a case-insensitive "contains" match over one or more columns.
func (q *Query) ILike(term string, columns ...string) *Query {
	if strings.TrimSpace(term) == "" || len(columns) == 0 {
		return q
	}
	ph := q.Arg("%" + term + "%")
	parts := make([]string, len(columns))
	for i, c := range columns {
		parts[i] = c + " ILIKE " + ph
	}
	return q.Where("(" + strings.Join(parts, " OR ") + ")")
}

// SQL renders the WHERE clause (including the leading keyword) or "".
func (q *Query) SQL() string {
	if len(q.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(q.where, " AND ")
}

// Args returns the bind values registered so far.
func (q *Query) Args() []any { return q.args }

// Set collects column assignments for UPDATE statements.
type Set struct {
	cols []string
	args []any
}

func (s *Set) Add(column string, value any) *Set {
	s.cols = append(s.cols, column)
	s.args = append(s.args, value)
	return s
}

func (s *Set) Empty() bool { return len(s.cols) == 0 }

// SQL renders "col1 = $1, col2 = $2, ..." starting at placeholder $offset+1.
func (s *Set) SQL(offset int) string {
	parts := make([]string, len(s.cols))
	for i, c := range s.cols {
		parts[i] = fmt.Sprintf("%s = $%d", c, offset+i+1)
	}
	return strings.Join(parts, ", ")
}

func (s *Set) Args() []any { return s.args }

// Columns renders a column list, optionally prefixed with a table alias.
func Columns(cols []string, prefix string) string {
	if prefix == "" {
		return strings.Join(cols, ", ")
	}
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = prefix + "." + c
	}
	return strings.Join(out, ", ")
}
