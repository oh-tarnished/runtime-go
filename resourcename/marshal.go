package resourcename

import (
	"fmt"
)

// MarshalResource converts a struct or pointer to struct into its resource
// name representation.
//
// The function resolves a template from the type, collects tagged field values,
// and fills placeholders in order.
//
// Example:
//
//	type User struct {
//	    _  struct{} `resource:"//bobthebuilder.com/users/{id}"`
//	    ID string   `resource:"id"`
//	}
//
//	rn, err := MarshalResource(&User{ID: "u42"})
func MarshalResource(v interface{}) (string, error) {
	structValue, err := requireStructValue(v)
	if err != nil {
		if v == nil {
			return "", fmt.Errorf("nil value")
		}
		return "", fmt.Errorf("MarshalResource: %w", err)
	}

	template, err := findTemplateString(structValue)
	if err != nil {
		return "", err
	}
	if template == "" {
		return "", fmt.Errorf("no resource template found on type %s", structValue.Type())
	}

	tpl, err := compileTemplate(template)
	if err != nil {
		return "", err
	}

	values := map[string]string{}
	if err := collectValues(structValue, "", values); err != nil {
		return "", err
	}
	return tpl.Generate(values)
}
