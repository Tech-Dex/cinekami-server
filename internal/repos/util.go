package repos

import (
	"fmt"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"cinekami-server/internal/model"
)

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func textVal(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func categoryToString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(x)
	}
}

// zeroTallies returns a map containing all allowed categories with 0 count.
func zeroTallies() map[string]int64 {
	m := make(map[string]int64, len(model.AllowedCategories))
	for k := range model.AllowedCategories {
		m[k] = 0
	}
	return m
}

// mergeTallies overlays counts from src into dst (which should already contain all categories).
func mergeTallies(dst map[string]int64, src map[string]int64) {
	for k, v := range src {
		dst[k] = v
	}
}

// anyToFloat64 converts a scanned SQL value to float64 when possible, else NaN.
func anyToFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case pgtype.Float8:
		if x.Valid {
			return x.Float64
		}
		return math.NaN()
	default:
		// Fallback: try parsing string
		s := fmt.Sprint(x)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return math.NaN()
		}
		return f
	}
}

// anyToInt64 converts a scanned SQL value to int64 when possible, else 0.
func anyToInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int32:
		return int64(x)
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	case string:
		if v, err := strconv.ParseInt(x, 10, 64); err == nil {
			return v
		}
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return int64(f)
		}
		return 0
	default:
		// try formatting then parse
		s := fmt.Sprint(x)
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
		return 0
	}
}
