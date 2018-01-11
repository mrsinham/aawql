package complexdriver

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Generic patterns in Google reports.
const (
	// Google can prefix a value by `auto:` or just return `auto` to symbolize an automatic strategy.
	auto      = "auto"
	autoValue = auto + ": "

	// Google uses `Excluded` to tag null value by context.
	excluded = "Excluded"

	// Google uses ` --` instead of an empty string to symbolize the fact that the field was never set.
	doubleDash = " --"

	// Google sometimes uses special value like `> 90%` or `< 10%`.
	almost10 = "< 10"
	almost90 = "> 90"
)

// PercentNullFloat64 represents a float64 that may be a percentage.
type PercentNullFloat64 struct {
	NullFloat64     sql.NullFloat64
	Almost, Percent bool
}

// Value implements the driver Valuer interface.
func (n PercentNullFloat64) Value() (driver.Value, error) {
	if !n.NullFloat64.Valid {
		return doubleDash, nil
	}
	var v string
	if n.Almost {
		if n.NullFloat64.Float64 > 90 {
			v = almost90
		} else {
			v = almost10
		}
	} else {
		v = strconv.FormatFloat(n.NullFloat64.Float64, 'f', 2, 64)
	}
	if n.Percent {
		return v + "%", nil
	}
	return v, nil
}

// Scan implements the driver Scanner interface.
func (n *PercentNullFloat64) Scan(d interface{}) (err error) {
	s, ok := d.(string)
	if !ok {
		return fmt.Errorf("unknow value %q", d)
	}
	if s == doubleDash {
		n.NullFloat64.Float64 = 0.0
		n.NullFloat64.Valid = false
		n.Almost = false
		n.Percent = false
		return
	}
	if n.Percent = strings.HasSuffix(s, "%"); n.Percent {
		s = strings.TrimSuffix(s, "%")
	}
	switch s {
	case almost10:
		// Sometimes, when it's less than 10, Google displays "< 10%".
		n.NullFloat64.Float64 = 9.999
		n.NullFloat64.Valid = true
		n.Almost = true
	case almost90:
		// Or "> 90%" when it is the opposite.
		n.NullFloat64.Float64 = 90.001
		n.NullFloat64.Valid = true
		n.Almost = true
	default:
		n.Almost = false
		s = strings.Replace(s, ",", "", -1)
		n.NullFloat64.Float64, err = strconv.ParseFloat(s, 64)
		if err == nil {
			n.NullFloat64.Valid = true
		}
	}
	return
}

// AutoExcludedNullInt64 represents a int64 that may be null or defined as auto valuer.
type AutoExcludedNullInt64 struct {
	NullInt64 sql.NullInt64
	Auto,
	Excluded bool
}

// Value implements the driver Valuer interface.
func (n AutoExcludedNullInt64) Value() (driver.Value, error) {
	var v string
	if n.Auto {
		if !n.NullInt64.Valid {
			return auto, nil
		}
		v = autoValue
	}
	if n.Excluded {
		return excluded, nil
	}
	if !n.NullInt64.Valid {
		return doubleDash, nil
	}
	v += strconv.FormatInt(n.NullInt64.Int64, 10)

	return v, nil
}

// Scan implements the driver Scanner interface.
func (n *AutoExcludedNullInt64) Scan(d interface{}) (err error) {
	s, ok := d.(string)
	if !ok {
		return fmt.Errorf("unknow value %q", d)
	}
	if s == doubleDash {
		return
	}
	if s == excluded {
		// Voluntary null by scope.
		n.Excluded = true
		return
	}
	if s, n.Auto = autoValued(s); s == "" {
		// Not set, null and automatic value.
		return
	}
	n.NullInt64.Int64, err = strconv.ParseInt(s, 10, 64)
	if err == nil {
		n.NullInt64.Valid = true
	}
	return
}

// autoValue trims prefixes `auto` and returns a cleaned string.
// Also indicates with the second parameter, if it's a automatic value or not.
func autoValued(s string) (v string, ok bool) {
	if ok = strings.HasPrefix(s, auto); !ok {
		// Not prefixed by auto keyword.
		v = s
		return
	}
	// Trims the prefix `auto: `
	if v = strings.TrimPrefix(s, autoValue); v == s {
		// Removes only `auto` as prefix
		v = strings.TrimPrefix(s, auto)
	}
	return
}

// Float64 represents a float64 that may be rounded by using its precision.
type Float64 struct {
	Float64   float64
	Precision int
}

// Value implements the driver Valuer interface.
func (n Float64) Value() (driver.Value, error) {
	return strconv.FormatFloat(n.Float64, 'f', n.Precision, 64), nil
}

// NullString represents a string that may be null.
type NullString struct {
	String string
	Valid  bool // Valid is true if String is not NULL
}

// Value implements the driver Valuer interface.
func (n NullString) Value() (driver.Value, error) {
	if !n.Valid {
		return doubleDash, nil
	}
	return n.String, nil
}

// Time represents a Time that may be not set.
type Time struct {
	Time   time.Time
	Layout string
}

// Value implements the driver Valuer interface.
func (n Time) Value() (driver.Value, error) {
	if n.Time.IsZero() {
		return doubleDash, nil
	}
	return n.Time.Format(n.Layout), nil
}
