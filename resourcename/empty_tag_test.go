package resourcename

import (
	"strings"
	"testing"
)

// TestEmptyTagMarshal verifies that empty resource tags cause marshal errors
func TestEmptyTagMarshal(t *testing.T) {
	type BadStruct struct {
		_    struct{} `resource:"//bobthebuilder.com/bad/{id}"`
		ID   string   `resource:""`
		Name string   `resource:"name"`
	}

	b := &BadStruct{ID: "test", Name: "test"}
	_, err := MarshalResource(b)
	if err == nil {
		t.Fatal("Expected error for empty resource tag, got nil")
	}
	if !strings.Contains(err.Error(), "empty resource tag") {
		t.Errorf("Expected error message to contain 'empty resource tag', got: %v", err)
	}
}

// TestEmptyTagUnmarshal verifies that empty resource tags cause unmarshal errors
func TestEmptyTagUnmarshal(t *testing.T) {
	type BadStruct struct {
		_    struct{} `resource:"//bobthebuilder.com/bad/{id}"`
		ID   string   `resource:""`
		Name string   `resource:"name"`
	}

	b := &BadStruct{}
	err := UnmarshalResource("//bobthebuilder.com/bad/test", b)
	if err == nil {
		t.Fatal("Expected error for empty resource tag, got nil")
	}
	if !strings.Contains(err.Error(), "empty resource tag") {
		t.Errorf("Expected error message to contain 'empty resource tag', got: %v", err)
	}
}

// TestEmptyTagNestedMarshal verifies that empty tags in nested structs cause errors
func TestEmptyTagNestedMarshal(t *testing.T) {
	type BadNested struct {
		City string `resource:""`
	}

	type BadParent struct {
		_      struct{}  `resource:"//bobthebuilder.com/parent/{nested.city}"`
		Nested BadNested `resource:"nested"`
	}

	p := &BadParent{Nested: BadNested{City: "NYC"}}
	_, err := MarshalResource(p)
	if err == nil {
		t.Fatal("Expected error for empty resource tag in nested struct, got nil")
	}
	if !strings.Contains(err.Error(), "empty resource tag") {
		t.Errorf("Expected error message to contain 'empty resource tag', got: %v", err)
	}
}

// TestEmptyTagNestedUnmarshal verifies that empty tags in nested structs cause unmarshal errors
func TestEmptyTagNestedUnmarshal(t *testing.T) {
	type BadNested struct {
		City string `resource:""`
	}

	type BadParent struct {
		_      struct{}  `resource:"//bobthebuilder.com/parent/{nested.city}"`
		Nested BadNested `resource:"nested"`
	}

	p := &BadParent{}
	err := UnmarshalResource("//bobthebuilder.com/parent/NYC", p)
	if err == nil {
		t.Fatal("Expected error for empty resource tag in nested struct, got nil")
	}
	if !strings.Contains(err.Error(), "empty resource tag") {
		t.Errorf("Expected error message to contain 'empty resource tag', got: %v", err)
	}
}
