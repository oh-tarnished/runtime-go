# Resoucename

`resourcename` is a small Go library for converting structs to and from templated
resource names.

It is useful when your IDs, paths, or cloud resource names follow a predictable
pattern such as:

```text
//bobthebuilder.com/users/{id}
//bobthebuilder.com/devices/{device_id}/sensors/{sensor_id}
```

The package keeps the public API intentionally small:

- `MarshalResource` turns a struct into a resource string.
- `UnmarshalResource` parses a resource string back into a struct.
- `Template` is the internal template representation used by the package.

## How templates are declared

There are two supported ways to declare the resource template for a type.

### 1. Template method

```go
type User struct {
	ID string `resource:"id"`
}

func (User) ResourceTemplate() string {
	return "//bobthebuilder.com/users/{id}"
}
```

### 2. Template tag

```go
type User struct {
	_  struct{} `resource:"//bobthebuilder.com/users/{id}"`
	ID string   `resource:"id"`
}
```

The tag form is commonly placed on an anonymous field, but any field tag with a
template-shaped value will work.

## Marshaling

`MarshalResource` walks the struct, reads the `resource:"..."` tags on exported
fields, and fills the placeholders in the template.

```go
type User struct {
	_     struct{} `resource:"//bobthebuilder.com/users/{id}"`
	ID    string   `resource:"id"`
	Active bool    `resource:"active"`
}

u := &User{ID: "u42", Active: true}
resource, err := resourcename.MarshalResource(u)
// resource == "//bobthebuilder.com/users/u42"
```

Notes:

- Strings, booleans, signed integers, unsigned integers, and pointer fields are
  supported.
- Types that implement `encoding.TextMarshaler` are marshaled through that
  interface.
- A field tag value of `""` is treated as an error.
- Placeholder values cannot contain `/`.

## Unmarshaling

`UnmarshalResource` parses a resource string and writes the extracted values
into the destination struct.

```go
var u User
err := resourcename.UnmarshalResource("//bobthebuilder.com/users/u42", &u)
// u.ID == "u42"
```

Notes:

- The target must be a non-nil pointer to a struct.
- Pointer fields are allocated as needed.
- Types that implement `encoding.TextUnmarshaler` are decoded through that
  interface.
- Nested structs are populated using dotted tags such as `resource:"address"`
  with template placeholders like `{address.city}`.

## Nested structs

Nested structs use dotted paths in both the template and the field tags.

```go
type Address struct {
	City string `resource:"city"`
	Zip  string `resource:"zip"`
}

type UserWithAddress struct {
	_       struct{} `resource:"//bobthebuilder.com/users/{id}/{address.city}/{address.zip}"`
	ID      string   `resource:"id"`
	Address Address  `resource:"address"`
}
```

With that structure:

- `Address.City` maps to `{address.city}`
- `Address.Zip` maps to `{address.zip}`

## Supported field kinds

The package supports these scalar kinds by default:

- `string`
- `bool`
- signed integers: `int`, `int8`, `int16`, `int32`, `int64`
- unsigned integers: `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `uintptr`

If a field type implements the standard text encoding interfaces, those take
priority over the built-in scalar conversions.

## Errors

Common errors include:

- nil input or nil target
- non-struct marshaling input
- non-pointer or non-struct unmarshal target
- missing resource template
- unsupported field kind
- malformed resource string that does not match the template

## Example

```go
package main

import (
	"fmt"

	"github.com/oh-tarnished/runtime-go/resourcename"
)

type Device struct {
	_        struct{} `resource:"//iot.example/devices/{device_id}/sensors/{sensor_id}"`
	DeviceID string   `resource:"device_id"`
	SensorID string   `resource:"sensor_id"`
}

func main() {
	device := &Device{DeviceID: "dev001", SensorID: "temp01"}

	resource, err := resourcename.MarshalResource(device)
	if err != nil {
		panic(err)
	}

	fmt.Println(resource)

	var decoded Device
	if err := resourcename.UnmarshalResource(resource, &decoded); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", decoded)
}
```
