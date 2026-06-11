package resourcename

import (
	"fmt"
	"regexp"
	"strings"
)

// Template holds a compiled resource pattern and the placeholder order it expects.
//
// A template is written as a resource string with placeholders in braces, such
// as //bobthebuilder.com/users/{id}/{region}. The placeholder order is preserved so
// parsing and generation stay deterministic.
type Template struct {
	raw          string
	regex        *regexp.Regexp
	placeholders []string
}

// compileTemplate converts a raw template string into a reusable Template.
func compileTemplate(raw string) (*Template, error) {
	placeholderPattern := regexp.MustCompile(`\{([^{}]+)\}`)
	matches := placeholderPattern.FindAllStringSubmatch(raw, -1)

	placeholders := make([]string, 0, len(matches))
	for _, match := range matches {
		placeholders = append(placeholders, match[1])
	}

	pattern := regexp.QuoteMeta(raw)
	for _, placeholder := range placeholders {
		pattern = strings.Replace(pattern, regexp.QuoteMeta("{"+placeholder+"}"), `([^/]+)`, 1)
	}

	compiled, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return nil, fmt.Errorf("invalid template pattern: %w", err)
	}

	return &Template{raw: raw, regex: compiled, placeholders: placeholders}, nil
}

// Generate builds a resource string from the provided placeholder values.
func (t *Template) Generate(values map[string]string) (string, error) {
	resource := t.raw
	for _, placeholder := range t.placeholders {
		value, ok := values[placeholder]
		if !ok {
			return "", fmt.Errorf("missing value for placeholder %q", placeholder)
		}
		if strings.Contains(value, "/") {
			return "", fmt.Errorf("value for %q contains '/'", placeholder)
		}
		resource = strings.Replace(resource, "{"+placeholder+"}", value, 1)
	}
	return resource, nil
}

// Parse extracts the placeholder values from a resource string.
func (t *Template) Parse(resource string) (map[string]string, error) {
	matches := t.regex.FindStringSubmatch(resource)
	if matches == nil {
		return nil, fmt.Errorf("resource %q does not match template %q", resource, t.raw)
	}
	if len(matches)-1 != len(t.placeholders) {
		return nil, fmt.Errorf("unexpected number of matches")
	}

	values := make(map[string]string, len(t.placeholders))
	for i, placeholder := range t.placeholders {
		values[placeholder] = matches[i+1]
	}
	return values, nil
}
