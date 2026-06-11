// Package resourcename converts Go structs to and from hierarchical resource names.
//
// The package is designed for readable, tag-driven mapping. A type can declare
// its resource template in one of two ways:
//
//   - by implementing ResourceTemplate() string
//   - by storing the template in a struct tag, usually on an anonymous field
//
// Example:
//
//	type User struct {
//	    _  struct{} `resource:"//bobthebuilder.com/users/{id}"`
//	    ID string   `resource:"id"`
//	}
//
// MarshalResource walks the struct, collects tagged field values, and fills the
// template placeholders.
//
// UnmarshalResource parses a resource string, extracts placeholder values, and
// writes them back into matching struct fields.
//
// Nested structs are supported with dotted paths such as {address.city}. The
// same dotted path is used on field tags, so a nested struct tagged with
// resource:"address" can fill {address.city} and {address.zip}.
//
// Supported scalar values include strings, booleans, signed integers, unsigned
// integers, and types that implement encoding.TextMarshaler/TextUnmarshaler.
//
// The package intentionally keeps the public API small: MarshalResource,
// UnmarshalResource, and Template are the main entry points.
package resourcename
