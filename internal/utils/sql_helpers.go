package utils

import (
	"database/sql"
	"strings"
	"time"
)

// NullStringToString convertit sql.NullString en string
func NullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// NullStringToPointer convertit sql.NullString en *string
func NullStringToPointer(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// NullInt64ToInt convertit sql.NullInt64 en int
func NullInt64ToInt(ni sql.NullInt64) int {
	if ni.Valid {
		return int(ni.Int64)
	}
	return 0
}

// NullInt64ToPointer convertit sql.NullInt64 en *int
func NullInt64ToPointer(ni sql.NullInt64) *int {
	if ni.Valid {
		val := int(ni.Int64)
		return &val
	}
	return nil
}

// NullFloat64ToFloat64 convertit sql.NullFloat64 en float64
func NullFloat64ToFloat64(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}

// NullFloat64ToPointer convertit sql.NullFloat64 en *float64
func NullFloat64ToPointer(nf sql.NullFloat64) *float64 {
	if nf.Valid {
		return &nf.Float64
	}
	return nil
}

// NullTimeToTime convertit sql.NullTime en time.Time
func NullTimeToTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}

// NullTimeToPointer convertit sql.NullTime en *time.Time
func NullTimeToPointer(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// NullBoolToBool convertit sql.NullBool en bool
func NullBoolToBool(nb sql.NullBool) bool {
	if nb.Valid {
		return nb.Bool
	}
	return false
}

// NullBoolToPointer convertit sql.NullBool en *bool
func NullBoolToPointer(nb sql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

func NullStringToStringArray(ns sql.NullString) []string {
	if !ns.Valid || ns.String == "" {
		return []string{}
	}

	// Enlever les accolades { }
	s := strings.Trim(ns.String, "{}")
	if s == "" {
		return []string{}
	}

	// Séparer par virgule
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
