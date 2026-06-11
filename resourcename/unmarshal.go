package resourcename

import "fmt"

// UnmarshalResource parses a resource string and populates the matching fields
// on a non-nil pointer to struct.
//
// Nested structs are walked using the same dotted path convention as marshaling.
// Fields that implement encoding.TextUnmarshaler are populated through that
// interface before falling back to built-in scalar conversion.
//
// Example:
//
//	type User struct {
//	    _  struct{} `resource:"//bobthebuilder.com/users/{id}"`
//	    ID string   `resource:"id"`
//	}
//
//	var user User
//	err := UnmarshalResource("//bobthebuilder.com/users/u42", &user)
func UnmarshalResource(resource string, v interface{}) error {
	structValue, err := requireStructPointer(v)
	if err != nil {
		if v == nil {
			return fmt.Errorf("nil target")
		}
		return fmt.Errorf("UnmarshalResource: %w", err)
	}

	template, err := findTemplateString(structValue)
	if err != nil {
		return err
	}
	if template == "" {
		return fmt.Errorf("no resource template found on type %s", structValue.Type())
	}

	tpl, err := compileTemplate(template)
	if err != nil {
		return err
	}

	parsed, err := tpl.Parse(resource)
	if err != nil {
		return err
	}

	return applyValues(structValue, "", parsed)
}
