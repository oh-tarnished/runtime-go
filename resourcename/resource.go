package resourcename

import (
	"fmt"
	"reflect"
	"strings"
)

// requireStructValue normalizes a struct value or pointer to struct into the
// underlying struct value used by MarshalResource.
func requireStructValue(v interface{}) (reflect.Value, error) {
	if v == nil {
		return reflect.Value{}, fmt.Errorf("nil value")
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return reflect.Value{}, fmt.Errorf("nil value")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expecting struct or pointer to struct")
	}

	return rv, nil
}

// requireStructPointer validates the input for UnmarshalResource and returns
// the underlying struct value so fields can be populated in place.
func requireStructPointer(v interface{}) (reflect.Value, error) {
	if v == nil {
		return reflect.Value{}, fmt.Errorf("nil target")
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return reflect.Value{}, fmt.Errorf("requires a non-nil pointer to a struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("requires pointer to struct")
	}

	return rv, nil
}

// findTemplateString discovers the resource template declared on a type.
//
// The lookup order is intentionally simple: first a ResourceTemplate method,
// then a struct tag that looks like a template string.
func findTemplateString(rv reflect.Value) string {
	if template, ok := templateFromMethod(rv); ok && template != "" {
		return template
	}

	if template, ok := templateFromTag(rv.Type()); ok {
		return template
	}

	return ""
}

// templateFromMethod checks both value and pointer receivers for a
// ResourceTemplate() string method.
func templateFromMethod(rv reflect.Value) (string, bool) {
	for _, candidate := range methodReceivers(rv) {
		if !candidate.IsValid() || !candidate.CanInterface() {
			continue
		}
		templater, ok := candidate.Interface().(interface{ ResourceTemplate() string })
		if !ok {
			continue
		}
		return strings.TrimSpace(templater.ResourceTemplate()), true
	}

	return "", false
}

// methodReceivers returns the value and, when useful, a pointer form of the
// value so both receiver styles can be checked.
func methodReceivers(rv reflect.Value) []reflect.Value {
	if !rv.IsValid() {
		return nil
	}

	receivers := []reflect.Value{rv}
	if rv.Kind() != reflect.Ptr {
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		receivers = append(receivers, ptr)
		return receivers
	}

	if rv.IsNil() {
		receivers = append(receivers, reflect.New(rv.Type().Elem()))
	}

	return receivers
}

// templateFromTag scans struct fields for a tag that looks like a template.
func templateFromTag(rt reflect.Type) (string, bool) {
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		tag, ok := sf.Tag.Lookup("resource")
		if !ok || tag == "" {
			continue
		}
		if looksLikeTemplate(tag) {
			return tag, true
		}
	}

	return "", false
}

// looksLikeTemplate is a small heuristic for identifying carrier tags that
// store the actual resource template.
func looksLikeTemplate(tag string) bool {
	return strings.HasPrefix(tag, "//") || (strings.Contains(tag, "{") && strings.Contains(tag, "}"))
}
