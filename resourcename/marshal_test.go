package resourcename

import (
	"testing"
)

type User struct {
	_    struct{} `resource:"//bobthebuilder.dev.com/User/{id}/{name}/{age}"`
	ID   string   `resource:"id"`
	Name string   `resource:"name"`
	Age  int      `resource:"age"`
}

type Product struct {
	_        struct{} `resource:"//store.com/products/{category}/{sku}"`
	Category string   `resource:"category"`
	SKU      string   `resource:"sku"`
}

type Device struct {
	_        struct{} `resource:"//iot.com/devices/{device_id}/sensors/{sensor_id}"`
	DeviceID string   `resource:"device_id"`
	SensorID string   `resource:"sensor_id"`
	Active   bool     `resource:"active"`
}

func TestMarshalResource(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{
			name: "basic user struct",
			input: &User{
				ID:   "u42",
				Name: "BobTheBuilder",
				Age:  29,
			},
			expected: "//bobthebuilder.dev.com/User/u42/BobTheBuilder/29",
			wantErr:  false,
		},
		{
			name: "user struct by value",
			input: User{
				ID:   "u100",
				Name: "John",
				Age:  35,
			},
			expected: "//bobthebuilder.dev.com/User/u100/John/35",
			wantErr:  false,
		},
		{
			name: "product struct",
			input: &Product{
				Category: "electronics",
				SKU:      "ABC123",
			},
			expected: "//store.com/products/electronics/ABC123",
			wantErr:  false,
		},
		{
			name: "device with bool",
			input: &Device{
				DeviceID: "dev001",
				SensorID: "temp01",
				Active:   true,
			},
			expected: "//iot.com/devices/dev001/sensors/temp01",
			wantErr:  false,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarshalResource(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("MarshalResource() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMarshalResource_EdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		u := &User{
			ID:   "",
			Name: "",
			Age:  0,
		}
		result, err := MarshalResource(u)
		if err != nil {
			t.Errorf("MarshalResource() error = %v", err)
		}
		expected := "//bobthebuilder.dev.com/User///0"
		if result != expected {
			t.Errorf("MarshalResource() = %v, want %v", result, expected)
		}
	})

	t.Run("special characters in name", func(t *testing.T) {
		u := &User{
			ID:   "u42",
			Name: "John-Doe",
			Age:  30,
		}
		result, err := MarshalResource(u)
		if err != nil {
			t.Errorf("MarshalResource() error = %v", err)
		}
		expected := "//bobthebuilder.dev.com/User/u42/John-Doe/30"
		if result != expected {
			t.Errorf("MarshalResource() = %v, want %v", result, expected)
		}
	})
}
