// Package conv to help type assertions and conversions.
package conv

import (
	"strconv"
)

var (
	// Error codes for store errors. This should match the codes
	// defined in the /simplesessions package exactly.
	ErrInvalidSession = &Err{code: 1, msg: "invalid session"}
	ErrFieldNotFound  = &Err{code: 2, msg: "field not found"}
	ErrAssertType     = &Err{code: 3, msg: "assertion failed"}
	ErrNil            = &Err{code: 4, msg: "nil returned"}
)

type Err struct {
	code int
	msg  string
}

func (e *Err) Error() string {
	return e.msg
}

func (e *Err) Code() int {
	return e.code
}

// Int converts interface to integer.
func Int(r interface{}, err error) (int, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case int:
		return r, nil
	case int64:
		x := int(r)
		if int64(x) != r {
			return 0, strconv.ErrRange
		}
		return x, nil
	case []byte:
		n, err := strconv.ParseInt(string(r), 10, 0)
		return int(n), err
	case string:
		n, err := strconv.ParseInt(r, 10, 0)
		return int(n), err
	case nil:
		return 0, ErrNil
	}

	return 0, ErrAssertType
}

// Int64 converts interface to Int64.
func Int64(r interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case int:
		return int64(r), nil
	case int64:
		return r, nil
	case []byte:
		n, err := strconv.ParseInt(string(r), 10, 64)
		return n, err
	case string:
		n, err := strconv.ParseInt(r, 10, 64)
		return n, err
	case nil:
		return 0, ErrNil
	}

	return 0, ErrAssertType
}

// UInt64 converts interface to UInt64.
func UInt64(r interface{}, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}

	switch r := r.(type) {
	case uint64:
		return r, err
	case int:
		if r < 0 {
			return 0, ErrAssertType
		}
		return uint64(r), nil
	case int64:
		if r < 0 {
			return 0, ErrAssertType
		}
		return uint64(r), nil
	case []byte:
		n, err := strconv.ParseUint(string(r), 10, 64)
		return n, err
	case string:
		n, err := strconv.ParseUint(r, 10, 64)
		return n, err
	case nil:
		return 0, ErrNil
	}

	return 0, ErrAssertType
}

// Float64 converts interface to Float64.
func Float64(r interface{}, err error) (float64, error) {
	if err != nil {
		return 0, err
	}
	switch r := r.(type) {
	case float64:
		return r, err
	case []byte:
		n, err := strconv.ParseFloat(string(r), 64)
		return n, err
	case string:
		n, err := strconv.ParseFloat(r, 64)
		return n, err
	case nil:
		return 0, ErrNil
	}
	return 0, ErrAssertType
}

// String converts interface to String.
func String(r interface{}, err error) (string, error) {
	if err != nil {
		return "", err
	}
	switch r := r.(type) {
	case []byte:
		return string(r), nil
	case string:
		return r, nil
	case nil:
		return "", ErrNil
	}
	return "", ErrAssertType
}

// Bytes converts interface to Bytes.
func Bytes(r interface{}, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	switch r := r.(type) {
	case []byte:
		return r, nil
	case string:
		return []byte(r), nil
	case nil:
		return nil, ErrNil
	}
	return nil, ErrAssertType
}

// Bool converts interface to Bool.
func Bool(r interface{}, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	switch r := r.(type) {
	case bool:
		return r, err
	// Very common in redis to reply int64 with 0 for bool flag.
	case int:
		return r != 0, nil
	case int64:
		return r != 0, nil
	case []byte:
		return strconv.ParseBool(string(r))
	case string:
		return strconv.ParseBool(r)
	case nil:
		return false, ErrNil
	}
	return false, ErrAssertType
}
