package resourcename

import (
	"fmt"
	"reflect"
)

// joinPath builds the dotted path used to match nested tags with nested
// placeholders such as {address.city}.
func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// isTemplateCarrierTag identifies the tag that stores the full resource
// template instead of a field mapping.
func isTemplateCarrierTag(tag string) bool {
	return looksLikeTemplate(tag)
}

// isTraversalStruct reports whether a value should be walked as a nested
// struct instead of converted to or from a scalar value.
func isTraversalStruct(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	typ := v.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return false
	}

	return !hasTextCodecType(typ)
}

func hasTextCodecType(typ reflect.Type) bool {
	if typ.Implements(textMarshalerType) || typ.Implements(textUnmarshalerType) {
		return true
	}
	if typ.Kind() != reflect.Ptr && reflect.PointerTo(typ).Implements(textMarshalerType) {
		return true
	}
	if typ.Kind() != reflect.Ptr && reflect.PointerTo(typ).Implements(textUnmarshalerType) {
		return true
	}
	return false
}

// collectValues walks a struct and produces the placeholder map used during
// marshaling.
func collectValues(structValue reflect.Value, prefix string, values map[string]string) error {
	structValue = dereferenceStruct(structValue)
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		sf := structType.Field(i)
		tag, ok := sf.Tag.Lookup("resource")
		if !ok {
			continue
		}
		if isTemplateCarrierTag(tag) {
			continue
		}
		if tag == "" {
			return fmt.Errorf("field %s has empty resource tag", sf.Name)
		}
		if !sf.IsExported() {
			continue
		}

		fieldValue := structValue.Field(i)
		path := joinPath(prefix, tag)

		if isTraversalStruct(fieldValue) {
			child, err := prepareNestedStruct(fieldValue)
			if err != nil {
				return fmt.Errorf("field %s: %w", sf.Name, err)
			}
			if err := collectValues(child, path, values); err != nil {
				return fmt.Errorf("field %s: %w", sf.Name, err)
			}
			continue
		}

		text, err := stringifyValue(fieldValue)
		if err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
		values[path] = text
	}

	return nil
}

// applyValues walks a struct and writes the parsed placeholder values back
// into matching fields.
func applyValues(structValue reflect.Value, prefix string, values map[string]string) error {
	structValue = dereferenceStruct(structValue)
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		sf := structType.Field(i)
		tag, ok := sf.Tag.Lookup("resource")
		if !ok {
			continue
		}
		if isTemplateCarrierTag(tag) {
			continue
		}
		if tag == "" {
			return fmt.Errorf("field %s has empty resource tag", sf.Name)
		}
		if !sf.IsExported() {
			continue
		}

		fieldValue := structValue.Field(i)
		path := joinPath(prefix, tag)

		if isTraversalStruct(fieldValue) {
			child, err := prepareNestedStruct(fieldValue)
			if err != nil {
				return fmt.Errorf("field %s: %w", sf.Name, err)
			}
			if err := applyValues(child, path, values); err != nil {
				return fmt.Errorf("field %s: %w", sf.Name, err)
			}
			continue
		}

		text, ok := values[path]
		if !ok {
			continue
		}

		if err := assignValue(fieldValue, text); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}

	return nil
}

func dereferenceStruct(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func prepareNestedStruct(v reflect.Value) (reflect.Value, error) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			if !v.CanSet() {
				return reflect.Value{}, fmt.Errorf("nil pointer")
			}
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected nested struct, got %s", v.Kind())
	}

	return v, nil
}
