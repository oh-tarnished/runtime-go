package resourcename

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
)

var (
	textMarshalerType   = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// stringifyValue converts a reflected value into the string form stored in a
// resource template.
func stringifyValue(v reflect.Value) (string, error) {
	if !v.IsValid() {
		return "", fmt.Errorf("invalid value")
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", fmt.Errorf("nil pointer")
		}
		return stringifyValue(v.Elem())
	}

	if text, ok, err := marshalText(v); ok || err != nil {
		return text, err
	}

	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	default:
		return "", fmt.Errorf("unsupported kind for marshaling: %s", v.Kind())
	}
}

func marshalText(v reflect.Value) (string, bool, error) {
	if v.CanInterface() {
		if marshaler, ok := v.Interface().(encoding.TextMarshaler); ok {
			data, err := marshaler.MarshalText()
			return string(data), true, err
		}
	}
	if v.CanAddr() {
		if marshaler, ok := v.Addr().Interface().(encoding.TextMarshaler); ok {
			data, err := marshaler.MarshalText()
			return string(data), true, err
		}
	}
	return "", false, nil
}

// assignValue converts a parsed string into the target field.
func assignValue(v reflect.Value, text string) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			if !v.CanSet() {
				return fmt.Errorf("cannot allocate pointer")
			}
			v.Set(reflect.New(v.Type().Elem()))
		}
		return assignValue(v.Elem(), text)
	}

	if v.CanAddr() {
		if unmarshaler, ok := v.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(text))
		}
	}
	if v.CanInterface() {
		if unmarshaler, ok := v.Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(text))
		}
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(text)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bits := 64
		switch v.Kind() {
		case reflect.Int8:
			bits = 8
		case reflect.Int16:
			bits = 16
		case reflect.Int32:
			bits = 32
		}
		number, err := strconv.ParseInt(text, 10, bits)
		if err != nil {
			return err
		}
		v.SetInt(number)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		bits := 64
		switch v.Kind() {
		case reflect.Uint8:
			bits = 8
		case reflect.Uint16:
			bits = 16
		case reflect.Uint32:
			bits = 32
		case reflect.Uintptr:
			bits = 0
		}
		number, err := strconv.ParseUint(text, 10, bits)
		if err != nil {
			return err
		}
		v.SetUint(number)
		return nil
	case reflect.Bool:
		value, err := strconv.ParseBool(text)
		if err != nil {
			return err
		}
		v.SetBool(value)
		return nil
	default:
		return fmt.Errorf("unsupported kind for unmarshaling: %s", v.Kind())
	}
}
