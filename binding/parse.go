package binding

import (
	"encoding"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
)

var ErrUnparsable = errors.New("unparsable value")

var intBitSizes = map[reflect.Kind]int{
	reflect.Int: 0, reflect.Int8: 8, reflect.Int16: 16,
	reflect.Int32: 32, reflect.Int64: 64,
}

var uintBitSizes = map[reflect.Kind]int{
	reflect.Uint: 0, reflect.Uint8: 8, reflect.Uint16: 16,
	reflect.Uint32: 32, reflect.Uint64: 64,
}

var floatBitSizes = map[reflect.Kind]int{
	reflect.Float32: 32, reflect.Float64: 64,
}


func ParseInto(ctx core.Context, f reflect.Value, value string, preSplit []string, p ParamFieldInfo, parsedQuery ...url.Values) (any, error) {
	switch p.Type.Kind() {
	case reflect.String:
		f.SetString(value)
		return value, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, intBitSizes[p.Type.Kind()])
		if err != nil {
			return nil, errors.New("invalid integer")
		}
		f.SetInt(v)
		return v, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, uintBitSizes[p.Type.Kind()])
		if err != nil {
			return nil, errors.New("invalid integer")
		}
		f.SetUint(v)
		return v, nil

	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, floatBitSizes[p.Type.Kind()])
		if err != nil {
			return nil, errors.New("invalid float")
		}
		f.SetFloat(v)
		return v, nil

	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return nil, errors.New("invalid boolean")
		}
		f.SetBool(v)
		return v, nil

	case reflect.Slice:
		var values []string
		if preSplit != nil {
			values = preSplit
		} else {
			if p.Explode {
				if len(parsedQuery) > 0 && parsedQuery[0] != nil {
					values = parsedQuery[0][p.Name]
				} else {
					u := ctx.URL()
					values = (&u).Query()[p.Name]
				}
			} else {
				values = strings.Split(value, ",")
			}
		}
		pv, err := parseSliceInto(f, values)
		if err != nil {
			if errors.Is(err, ErrUnparsable) {
				break
			}
			return nil, err
		}
		return pv, nil
	default:
		// unsupported type
	}

	switch f.Type() {
	case timeType:
		t, err := time.Parse(p.TimeFormat, value)
		if err != nil {
			return nil, errors.New("invalid date/time for format " + p.TimeFormat)
		}
		f.Set(reflect.ValueOf(t))
		return value, nil
	case urlType:
		u, err := url.Parse(value)
		if err != nil {
			return nil, errors.New("invalid url.URL value")
		}
		f.Set(reflect.ValueOf(*u))
		return value, nil
	}

	// Last resort: use the encoding.TextUnmarshaler interface.
	if fn, ok := f.Addr().Interface().(encoding.TextUnmarshaler); ok {
		if err := fn.UnmarshalText([]byte(value)); err != nil {
			return nil, errors.New("invalid value: " + err.Error())
		}
		return value, nil
	}

	return nil, fmt.Errorf("unsupported param type %s", p.Type.String())
}


func parseAndSetSlice[T any](f reflect.Value, values []string, parse func(string) (T, error), setter func(reflect.Value, T), errMsg string) (any, error) {
	vs, err := parseArrElement(values, parse)
	if err != nil {
		return nil, errors.New(errMsg)
	}
	slice := reflect.MakeSlice(f.Type(), len(vs), len(vs))
	for i, v := range vs {
		setter(slice.Index(i), v)
	}
	f.Set(slice)
	return slice.Interface(), nil
}

func parseArrElement[T any](values []string, parse func(string) (T, error)) ([]T, error) {
	result := make([]T, 0, len(values))
	for i := range values {
		v, err := parse(values[i])
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

func parseScalar(f reflect.Value, value string) error {
	switch f.Kind() {
	case reflect.String:
		f.SetString(value)
	case reflect.Interface:
		f.Set(reflect.ValueOf(value))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, intBitSizes[f.Kind()])
		if err != nil {
			return errors.New("invalid integer")
		}
		f.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, uintBitSizes[f.Kind()])
		if err != nil {
			return errors.New("invalid integer")
		}
		f.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, floatBitSizes[f.Kind()])
		if err != nil {
			return errors.New("invalid float")
		}
		f.SetFloat(v)
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return errors.New("invalid boolean")
		}
		f.SetBool(v)
	default:
		return ErrUnparsable
	}
	return nil
}

func parseSliceInto(f reflect.Value, values []string) (any, error) {
	elemKind := f.Type().Elem().Kind()

	switch elemKind {
	case reflect.String:
		if f.Type() == stringSliceType {
			f.Set(reflect.ValueOf(values))
		} else {
			// Change element type to support slice of string subtypes (enums).
			enumValues := reflect.New(f.Type()).Elem()
			for _, val := range values {
				enumVal := reflect.New(f.Type().Elem()).Elem()
				enumVal.SetString(val)
				enumValues.Set(reflect.Append(enumValues, enumVal))
			}
			f.Set(enumValues)
		}
		return values, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bitSize := intBitSizes[elemKind]
		return parseAndSetSlice(f, values, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, bitSize)
		}, func(v reflect.Value, val int64) { v.SetInt(val) }, "invalid integer")

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bitSize := uintBitSizes[elemKind]
		return parseAndSetSlice(f, values, func(s string) (uint64, error) {
			return strconv.ParseUint(s, 10, bitSize)
		}, func(v reflect.Value, val uint64) { v.SetUint(val) }, "invalid integer")

	case reflect.Float32, reflect.Float64:
		bitSize := floatBitSizes[elemKind]
		return parseAndSetSlice(f, values, func(s string) (float64, error) {
			return strconv.ParseFloat(s, bitSize)
		}, func(v reflect.Value, val float64) { v.SetFloat(val) }, "invalid float")

	case reflect.Bool:
		return parseAndSetSlice(f, values, strconv.ParseBool,
			func(v reflect.Value, val bool) { v.SetBool(val) }, "invalid boolean")
	default:
		// unsupported type
	}

	return nil, ErrUnparsable
}

func setFieldValue(f reflect.Value, value string) error {
	err := parseScalar(f, value)
	if errors.Is(err, ErrUnparsable) {
		return errors.New("unsupported type")
	}
	return err
}
