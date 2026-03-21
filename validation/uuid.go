// Borrowed from the Google UUID package:
// https://github.com/google/uuid/blob/v1.6.0/uuid.go
// Copyright (c) 2009,2014 Google Inc. All rights reserved.
// BSD licensed.

// Package validation provides format validation functions used by the neoma
// schema validation engine.
package validation

import (
	"errors"
	"fmt"
	"strings"
)

type invalidLengthError struct{ len int }

func (err invalidLengthError) Error() string {
	return fmt.Sprintf("invalid UUID length: %d", err.len)
}

// ValidateUUID checks whether s is a valid UUID in standard, URN, bracketed,
// or hex-only format, returning an error if the format is invalid.
func ValidateUUID(s string) error {
	switch len(s) {
	case 36:

	case 36 + 9:
		if !strings.EqualFold(s[:9], "urn:uuid:") {
			return fmt.Errorf("invalid urn prefix: %q", s[:9])
		}
		s = s[9:]

	case 36 + 2:
		if s[0] != '{' || s[len(s)-1] != '}' {
			return errors.New("invalid bracketed UUID format")
		}
		s = s[1 : len(s)-1]

	case 32:
		for i := 0; i < len(s); i += 2 {
			_, ok := xtob(s[i], s[i+1])
			if !ok {
				return errors.New("invalid UUID format")
			}
		}

	default:
		return invalidLengthError{len(s)}
	}

	if len(s) == 36 {
		if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
			return errors.New("invalid UUID format")
		}
		for _, x := range []int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34} {
			if _, ok := xtob(s[x], s[x+1]); !ok {
				return errors.New("invalid UUID format")
			}
		}
	}

	return nil
}

var xvalues = [256]byte{
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 255, 255, 255, 255, 255, 255,
	255, 10, 11, 12, 13, 14, 15, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 10, 11, 12, 13, 14, 15, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
}

func xtob(x1, x2 byte) (byte, bool) { //nolint:unparam // borrowed from google/uuid
	b1 := xvalues[x1]
	b2 := xvalues[x2]
	return (b1 << 4) | b2, b1 != 255 && b2 != 255
}
