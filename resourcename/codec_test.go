package resourcename

import (
	"strings"
	"testing"
)

type methodTemplateUser struct {
	ID string `resource:"id"`
}

func (methodTemplateUser) ResourceTemplate() string {
	return "//bobthebuilder.com/method-users/{id}"
}

type token string

func (t token) MarshalText() ([]byte, error) {
	return []byte("token:" + string(t)), nil
}

func (t *token) UnmarshalText(data []byte) error {
	*t = token(strings.TrimPrefix(string(data), "token:"))
	return nil
}

type tokenResource struct {
	_     struct{} `resource:"//bobthebuilder.com/tokens/{value}"`
	Value token    `resource:"value"`
}

func TestMarshalResourceUsesResourceTemplateMethod(t *testing.T) {
	resource, err := MarshalResource(&methodTemplateUser{ID: "u42"})
	if err != nil {
		t.Fatalf("MarshalResource() error = %v", err)
	}
	if resource != "//bobthebuilder.com/method-users/u42" {
		t.Fatalf("MarshalResource() = %v, want %v", resource, "//bobthebuilder.com/method-users/u42")
	}
}

func TestTextMarshalerFieldsRoundTrip(t *testing.T) {
	original := &tokenResource{Value: token("alpha")}
	resource, err := MarshalResource(original)
	if err != nil {
		t.Fatalf("MarshalResource() error = %v", err)
	}
	if resource != "//bobthebuilder.com/tokens/token:alpha" {
		t.Fatalf("MarshalResource() = %v, want %v", resource, "//bobthebuilder.com/tokens/token:alpha")
	}

	decoded := &tokenResource{}
	if err := UnmarshalResource(resource, decoded); err != nil {
		t.Fatalf("UnmarshalResource() error = %v", err)
	}
	if decoded.Value != token("alpha") {
		t.Fatalf("decoded.Value = %v, want %v", decoded.Value, token("alpha"))
	}
}
