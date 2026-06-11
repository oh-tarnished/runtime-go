package resourcename

import (
	"testing"
)

func TestUnmarshalResource(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		target   interface{}
		check    func(t *testing.T, v interface{})
		wantErr  bool
	}{
		{
			name:     "basic user unmarshal",
			resource: "//bobthebuilder.dev.com/User/u42/BobTheBuilder/29",
			target:   &User{},
			check: func(t *testing.T, v interface{}) {
				u := v.(*User)
				if u.ID != "u42" {
					t.Errorf("ID = %v, want u42", u.ID)
				}
				if u.Name != "BobTheBuilder" {
					t.Errorf("Name = %v, want BobTheBuilder", u.Name)
				}
				if u.Age != 29 {
					t.Errorf("Age = %v, want 29", u.Age)
				}
			},
			wantErr: false,
		},
		{
			name:     "product unmarshal",
			resource: "//store.com/products/electronics/ABC123",
			target:   &Product{},
			check: func(t *testing.T, v interface{}) {
				p := v.(*Product)
				if p.Category != "electronics" {
					t.Errorf("Category = %v, want electronics", p.Category)
				}
				if p.SKU != "ABC123" {
					t.Errorf("SKU = %v, want ABC123", p.SKU)
				}
			},
			wantErr: false,
		},
		{
			name:     "device unmarshal",
			resource: "//iot.com/devices/dev001/sensors/temp01",
			target:   &Device{},
			check: func(t *testing.T, v interface{}) {
				d := v.(*Device)
				if d.DeviceID != "dev001" {
					t.Errorf("DeviceID = %v, want dev001", d.DeviceID)
				}
				if d.SensorID != "temp01" {
					t.Errorf("SensorID = %v, want temp01", d.SensorID)
				}
			},
			wantErr: false,
		},
		{
			name:     "invalid resource format",
			resource: "invalid-resource",
			target:   &User{},
			check:    nil,
			wantErr:  true,
		},
		{
			name:     "nil target",
			resource: "//bobthebuilder.dev.com/User/u42/BobTheBuilder/29",
			target:   nil,
			check:    nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnmarshalResource(tt.resource, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, tt.target)
			}
		})
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name: "user round trip",
			input: &User{
				ID:   "u42",
				Name: "BobTheBuilder",
				Age:  29,
			},
		},
		{
			name: "product round trip",
			input: &Product{
				Category: "books",
				SKU:      "ISBN-123",
			},
		},
		{
			name: "device round trip",
			input: &Device{
				DeviceID: "dev999",
				SensorID: "humidity02",
				Active:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			resource, err := MarshalResource(tt.input)
			if err != nil {
				t.Fatalf("MarshalResource() error = %v", err)
			}

			// Unmarshal
			var target interface{}
			switch tt.input.(type) {
			case *User:
				target = &User{}
			case *Product:
				target = &Product{}
			case *Device:
				target = &Device{}
			}

			err = UnmarshalResource(resource, target)
			if err != nil {
				t.Fatalf("UnmarshalResource() error = %v", err)
			}

			// Marshal again and compare
			resource2, err := MarshalResource(target)
			if err != nil {
				t.Fatalf("MarshalResource() second time error = %v", err)
			}

			if resource != resource2 {
				t.Errorf("Round trip failed: %v != %v", resource, resource2)
			}
		})
	}
}
