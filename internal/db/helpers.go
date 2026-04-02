// Package db provides database connectivity and operations.
package db

import (
	"database/sql"
)

// ToNullString converts a string to sql.NullString.
func ToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// ToNullInt64 converts an int64 to sql.NullInt64.
func ToNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: true}
}

// ToNullFloat64 converts a float64 to sql.NullFloat64.
func ToNullFloat64(f float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: f, Valid: true}
}

// NullStringValue returns the string value or empty string if null.
func NullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// NullInt64Value returns the int64 value or 0 if null.
func NullInt64Value(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// NullFloat64Value returns the float64 value or 0 if null.
func NullFloat64Value(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}
